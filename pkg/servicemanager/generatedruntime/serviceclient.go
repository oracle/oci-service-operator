/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/oracle/oci-go-sdk/v65/common"
	databasesdk "github.com/oracle/oci-go-sdk/v65/database"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/errorutil"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const defaultRequeueDuration = time.Minute

var errResourceNotFound = errors.New("generated runtime resource not found")

var (
	passwordSourceType         = reflect.TypeOf(shared.PasswordSource{})
	usernameSourceType         = reflect.TypeOf(shared.UsernameSource{})
	autonomousDatabaseBaseType = reflect.TypeOf((*databasesdk.CreateAutonomousDatabaseBase)(nil)).Elem()
)

type createContextKey string

const skipExistingBeforeCreateContextKey createContextKey = "generatedruntime/skip-existing-before-create"

type Operation struct {
	NewRequest func() any
	Call       func(context.Context, any) (any, error)
	Fields     []RequestField
}

type RequestField struct {
	FieldName        string
	RequestName      string
	Contribution     string
	PreferResourceID bool
	LookupPaths      []string
}

type Hook struct {
	Helper     string
	EntityType string
	Action     string
}

type FollowUpSemantics struct {
	Strategy string
	Hooks    []Hook
}

type HookSet struct {
	Create []Hook
	Update []Hook
	Delete []Hook
}

type LifecycleSemantics struct {
	ProvisioningStates []string
	UpdatingStates     []string
	ActiveStates       []string
}

type DeleteSemantics struct {
	Policy         string
	PendingStates  []string
	TerminalStates []string
}

type ListSemantics struct {
	ResponseItemsField string
	MatchFields        []string
}

type MutationSemantics struct {
	Mutable       []string
	ForceNew      []string
	ConflictsWith map[string][]string
}

type AuxiliaryOperation struct {
	Phase            string
	MethodName       string
	RequestTypeName  string
	ResponseTypeName string
}

type UnsupportedSemantic struct {
	Category      string
	StopCondition string
}

type Semantics struct {
	FormalService       string
	FormalSlug          string
	StatusProjection    string
	SecretSideEffects   string
	FinalizerPolicy     string
	Lifecycle           LifecycleSemantics
	Delete              DeleteSemantics
	List                *ListSemantics
	Mutation            MutationSemantics
	Hooks               HookSet
	CreateFollowUp      FollowUpSemantics
	UpdateFollowUp      FollowUpSemantics
	DeleteFollowUp      FollowUpSemantics
	AuxiliaryOperations []AuxiliaryOperation
	Unsupported         []UnsupportedSemantic
}

type Config[T any] struct {
	Kind             string
	SDKName          string
	Log              loggerutil.OSOKLogger
	CredentialClient credhelper.CredentialClient
	InitError        error
	Semantics        *Semantics
	BuildCreateBody  func(context.Context, T, string) (any, error)

	Create *Operation
	Get    *Operation
	List   *Operation
	Update *Operation
	Delete *Operation
}

type ServiceClient[T any] struct {
	config Config[T]
}

type readPhase string

type createOrUpdateState struct {
	currentID    string
	liveResponse any
}

type readResourceState struct {
	values     map[string]any
	readID     string
	listValues map[string]any
	listID     string
}

const (
	readPhaseObserve readPhase = "observe"
	readPhaseCreate  readPhase = "create"
	readPhaseUpdate  readPhase = "update"
	readPhaseDelete  readPhase = "delete"
)

func NewServiceClient[T any](cfg Config[T]) ServiceClient[T] {
	if err := validateFormalSemantics(cfg.Kind, cfg.Semantics); err != nil {
		cfg.InitError = errors.Join(cfg.InitError, err)
	}
	return ServiceClient[T]{config: cfg}
}

func WithSkipExistingBeforeCreate(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, skipExistingBeforeCreateContextKey, true)
}

func (c ServiceClient[T]) CreateOrUpdate(ctx context.Context, resource T, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if response, err, handled := c.validateCreateOrUpdateRequest(resource); handled {
		return response, err
	}

	namespace := resourceNamespace(resource, req.Namespace)
	state, err := c.prepareCreateOrUpdateState(ctx, resource)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if err := c.validateMutationPolicy(resource, state.currentID != "", state.liveResponse); err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if state.currentID != "" {
		return c.reconcileExistingResource(ctx, resource, state, namespace)
	}
	return c.createOrReadResource(ctx, resource, namespace)
}

func (c ServiceClient[T]) validateCreateOrUpdateRequest(resource T) (servicemanager.OSOKResponse, error, bool) {
	if c.config.InitError != nil {
		response, err := c.failCreateOrUpdate(resource, c.config.InitError)
		return response, err, true
	}
	if _, err := resourceStruct(resource); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err, true
	}
	return servicemanager.OSOKResponse{}, nil, false
}

func (c ServiceClient[T]) prepareCreateOrUpdateState(ctx context.Context, resource T) (createOrUpdateState, error) {
	currentID, existingResponse, resolvedBeforeCreate, err := c.resolveCurrentResource(ctx, resource)
	if err != nil {
		return createOrUpdateState{}, err
	}

	currentID, liveResponse, err := c.loadLiveMutationResponse(ctx, resource, currentID, existingResponse, resolvedBeforeCreate)
	if err != nil {
		return createOrUpdateState{}, err
	}

	return createOrUpdateState{currentID: currentID, liveResponse: liveResponse}, nil
}

func (c ServiceClient[T]) resolveCurrentResource(ctx context.Context, resource T) (string, any, bool, error) {
	currentID := c.currentID(resource)
	existingResponse, err := c.resolveExistingBeforeCreate(ctx, resource)
	if err != nil {
		return "", nil, false, err
	}

	currentID, resolvedBeforeCreate := c.resolveTrackedCurrentID(resource, currentID, existingResponse)
	return currentID, existingResponse, resolvedBeforeCreate, nil
}

func (c ServiceClient[T]) resolveTrackedCurrentID(resource T, currentID string, existingResponse any) (string, bool) {
	originalCurrentID := currentID
	resolvedBeforeCreate := currentID == "" && existingResponse != nil
	if c.shouldResolveExistingBeforeCreate() && c.usesStatusOnlyCurrentID(resource, currentID) {
		currentID = ""
		resolvedBeforeCreate = existingResponse != nil && responseID(existingResponse) != originalCurrentID
	}
	if currentID == "" && existingResponse != nil {
		currentID = responseID(existingResponse)
	}
	return currentID, resolvedBeforeCreate
}

func (c ServiceClient[T]) loadLiveMutationResponse(ctx context.Context, resource T, currentID string, existingResponse any, resolvedBeforeCreate bool) (string, any, error) {
	liveResponse := existingResponse
	if currentID == "" || !c.requiresLiveMutationAssessment() {
		c.mergeLiveResponseIntoStatus(resource, currentID, liveResponse)
		return currentID, liveResponse, nil
	}

	forceLiveGet := resolvedBeforeCreate && c.config.Get != nil
	if liveResponse == nil || forceLiveGet {
		var err error
		currentID, liveResponse, err = c.readMutationAssessmentResponse(ctx, resource, currentID, forceLiveGet)
		if err != nil {
			return "", nil, err
		}
	}

	c.mergeLiveResponseIntoStatus(resource, currentID, liveResponse)
	return currentID, liveResponse, nil
}

func (c ServiceClient[T]) readMutationAssessmentResponse(ctx context.Context, resource T, currentID string, forceLiveGet bool) (string, any, error) {
	response, err := c.readResourceForMutationValidation(ctx, resource, currentID, forceLiveGet)
	switch {
	case err == nil:
		return currentID, response, nil
	case !errors.Is(err, errResourceNotFound):
		return currentID, nil, err
	case forceLiveGet:
		return "", nil, nil
	default:
		return currentID, nil, nil
	}
}

func (c ServiceClient[T]) mergeLiveResponseIntoStatus(resource T, currentID string, liveResponse any) {
	if currentID != "" && liveResponse != nil {
		_ = mergeResponseIntoStatus(resource, liveResponse)
	}
}

func (c ServiceClient[T]) reconcileExistingResource(ctx context.Context, resource T, state createOrUpdateState, namespace string) (servicemanager.OSOKResponse, error) {
	shouldUpdate, err := c.shouldInvokeUpdate(resource, state.liveResponse)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if shouldUpdate {
		return c.updateExistingResource(ctx, resource, state.currentID, namespace, state.liveResponse)
	}
	return c.observeExistingResource(ctx, resource, state.currentID, state.liveResponse)
}

func (c ServiceClient[T]) updateExistingResource(ctx context.Context, resource T, currentID string, namespace string, currentResponse any) (servicemanager.OSOKResponse, error) {
	options := c.requestBuildOptions(ctx, namespace)
	options.CurrentResponse = currentResponse

	response, err := c.invoke(ctx, c.config.Update, resource, currentID, options)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}

	response, err = c.followUpAfterWrite(ctx, resource, currentID, response, "update")
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	return c.applySuccess(resource, response, shared.Updating)
}

func (c ServiceClient[T]) observeExistingResource(ctx context.Context, resource T, currentID string, liveResponse any) (servicemanager.OSOKResponse, error) {
	response := liveResponse
	if response == nil && (c.config.Get != nil || c.config.List != nil) {
		var err error
		response, err = c.readResource(ctx, resource, currentID, readPhaseObserve)
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
	}
	return c.applySuccess(resource, response, shared.Active)
}

func (c ServiceClient[T]) createOrReadResource(ctx context.Context, resource T, namespace string) (servicemanager.OSOKResponse, error) {
	if c.config.Create != nil {
		response, err := c.invoke(ctx, c.config.Create, resource, "", c.requestBuildOptions(ctx, namespace))
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}

		followUp, err := c.followUpAfterWrite(ctx, resource, responseID(response), response, "create")
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
		return c.applySuccess(resource, followUp, shared.Provisioning)
	}

	response, err := c.readResource(ctx, resource, "", readPhaseObserve)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	return c.applySuccess(resource, response, shared.Active)
}

func (c ServiceClient[T]) failCreateOrUpdate(resource T, err error) (servicemanager.OSOKResponse, error) {
	return servicemanager.OSOKResponse{IsSuccessful: false}, c.markFailure(resource, err)
}

func (c ServiceClient[T]) requestBuildOptions(ctx context.Context, namespace string) requestBuildOptions {
	return requestBuildOptions{
		Context:          ctx,
		CredentialClient: c.config.CredentialClient,
		Namespace:        namespace,
	}
}

func (c ServiceClient[T]) Delete(ctx context.Context, resource T) (bool, error) {
	if err := c.validateDeleteRequest(resource); err != nil {
		return false, err
	}
	if c.config.Semantics != nil {
		return c.deleteWithSemantics(ctx, resource)
	}
	if c.config.Delete == nil {
		c.markDeleted(resource, "OCI delete is not supported for this generated resource")
		return true, nil
	}

	currentID := c.currentID(resource)
	if currentID == "" {
		c.markDeleted(resource, "OCI resource identifier is not recorded")
		return true, nil
	}
	if deleted, err := c.invokeDeleteOperation(ctx, resource, currentID); deleted || err != nil {
		return deleted, err
	}
	return c.confirmDeleteWithoutSemantics(ctx, resource, currentID)
}

func (c ServiceClient[T]) validateDeleteRequest(resource T) error {
	if c.config.InitError != nil {
		return c.config.InitError
	}
	_, err := resourceStruct(resource)
	return err
}

func (c ServiceClient[T]) invokeDeleteOperation(ctx context.Context, resource T, currentID string) (bool, error) {
	if _, err := c.invoke(ctx, c.config.Delete, resource, currentID, requestBuildOptions{}); err != nil {
		if isNotFound(err) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (c ServiceClient[T]) confirmDeleteWithoutSemantics(ctx context.Context, resource T, currentID string) (bool, error) {
	if c.config.Get == nil && c.config.List == nil {
		c.markDeleted(resource, "OCI delete request accepted")
		return true, nil
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		if isNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, err
	}

	_ = mergeResponseIntoStatus(resource, response)
	c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
	return false, nil
}

func (c ServiceClient[T]) followUpAfterWrite(ctx context.Context, resource T, preferredID string, response any, phase string) (any, error) {
	if !c.requiresWriteFollowUp(phase) {
		return response, nil
	}
	if c.config.Get == nil && c.config.List == nil {
		if c.config.Semantics != nil {
			return nil, fmt.Errorf("%s formal semantics require %s follow-up without a readable OCI operation", c.config.Kind, phase)
		}
		return response, nil
	}

	refreshed, err := c.readResource(ctx, resource, preferredID, phaseReadPhase(phase))
	if err == nil {
		return refreshed, nil
	}
	if phase == "create" && errors.Is(err, errResourceNotFound) {
		return response, nil
	}
	return nil, err
}

func (c ServiceClient[T]) requiresWriteFollowUp(phase string) bool {
	if c.config.Semantics == nil {
		return c.config.Get != nil || c.config.List != nil
	}

	switch phase {
	case "create":
		return c.config.Semantics.CreateFollowUp.Strategy == "read-after-write"
	case "update":
		return c.config.Semantics.UpdateFollowUp.Strategy == "read-after-write"
	default:
		return false
	}
}

func phaseReadPhase(phase string) readPhase {
	switch phase {
	case "create":
		return readPhaseCreate
	case "update":
		return readPhaseUpdate
	case "delete":
		return readPhaseDelete
	default:
		return readPhaseObserve
	}
}

func (c ServiceClient[T]) deleteWithSemantics(ctx context.Context, resource T) (bool, error) {
	semantics, err := c.semanticDeleteConfig()
	if err != nil {
		return false, err
	}

	currentID, err := c.resolveDeleteID(ctx, resource)
	if err != nil {
		if errors.Is(err, errResourceNotFound) {
			c.markDeleted(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, err
	}
	if deleted, err, handled := c.confirmDeleteIfAlreadyPending(ctx, resource, currentID, semantics); handled {
		return deleted, err
	}
	deleted, err := c.invokeDeleteOperation(ctx, resource, currentID)
	if deleted {
		return true, nil
	}
	if err != nil && !c.shouldConfirmDeleteAfterError(err) {
		return false, err
	}
	return c.confirmDeleteWithSemantics(ctx, resource, currentID, semantics)
}

func (c ServiceClient[T]) confirmDeleteIfAlreadyPending(ctx context.Context, resource T, currentID string, semantics *Semantics) (bool, error, bool) {
	if semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		return false, nil, false
	}
	if c.config.Get == nil && c.config.List == nil {
		return false, nil, false
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		if isNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil, true
		}
		return false, nil, false
	}

	lifecycleState := strings.ToUpper(responseLifecycleState(response))
	if !containsString(semantics.Delete.PendingStates, lifecycleState) &&
		!containsString(semantics.Delete.TerminalStates, lifecycleState) {
		return false, nil, false
	}

	_ = mergeResponseIntoStatus(resource, response)
	deleted, err := c.applyDeletePolicy(resource, response, semantics)
	return deleted, err, true
}

func (c ServiceClient[T]) shouldConfirmDeleteAfterError(err error) bool {
	if err == nil || !isConflict(err) {
		return false
	}
	if c.config.Semantics == nil || c.config.Semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		return false
	}
	return c.config.Get != nil || c.config.List != nil
}

func (c ServiceClient[T]) semanticDeleteConfig() (*Semantics, error) {
	semantics := c.config.Semantics
	if semantics == nil {
		return nil, fmt.Errorf("%s formal semantics are not configured", c.config.Kind)
	}
	if c.config.Delete == nil || semantics.Delete.Policy == "not-supported" {
		return nil, fmt.Errorf("%s formal semantics mark delete confirmation as %q", c.config.Kind, semantics.Delete.Policy)
	}
	return semantics, nil
}

func (c ServiceClient[T]) confirmDeleteWithSemantics(ctx context.Context, resource T, currentID string, semantics *Semantics) (bool, error) {
	if semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		c.markDeleted(resource, "OCI delete request accepted")
		return true, nil
	}
	if c.config.Get == nil && c.config.List == nil {
		return false, fmt.Errorf("%s formal delete confirmation requires a readable OCI operation", c.config.Kind)
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		if isNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.markDeleted(resource, "OCI resource deleted")
			return true, nil
		}
		return false, err
	}

	_ = mergeResponseIntoStatus(resource, response)
	return c.applyDeletePolicy(resource, response, semantics)
}

func (c ServiceClient[T]) applyDeletePolicy(resource T, response any, semantics *Semantics) (bool, error) {
	lifecycleState := strings.ToUpper(responseLifecycleState(response))
	switch semantics.Delete.Policy {
	case "best-effort":
		return c.bestEffortDeleteOutcome(resource, lifecycleState, semantics)
	case "required":
		return c.requiredDeleteOutcome(resource, lifecycleState, semantics)
	default:
		return false, fmt.Errorf("%s formal delete confirmation policy %q is not supported", c.config.Kind, semantics.Delete.Policy)
	}
}

func (c ServiceClient[T]) bestEffortDeleteOutcome(resource T, lifecycleState string, semantics *Semantics) (bool, error) {
	if lifecycleState == "" ||
		containsString(semantics.Delete.PendingStates, lifecycleState) ||
		containsString(semantics.Delete.TerminalStates, lifecycleState) {
		c.markDeleted(resource, "OCI delete request accepted")
		return true, nil
	}

	c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
	return false, nil
}

func (c ServiceClient[T]) requiredDeleteOutcome(resource T, lifecycleState string, semantics *Semantics) (bool, error) {
	switch {
	case containsString(semantics.Delete.TerminalStates, lifecycleState):
		c.markDeleted(resource, "OCI resource deleted")
		return true, nil
	case lifecycleState == "" || containsString(semantics.Delete.PendingStates, lifecycleState):
		c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
		return false, nil
	default:
		return false, fmt.Errorf("%s delete confirmation returned unexpected lifecycle state %q", c.config.Kind, lifecycleState)
	}
}

func (c ServiceClient[T]) resolveDeleteID(ctx context.Context, resource T) (string, error) {
	currentID := c.currentID(resource)
	if currentID != "" {
		return currentID, nil
	}

	if c.config.Get == nil && c.config.List == nil {
		return "", errResourceNotFound
	}

	response, err := c.readResource(ctx, resource, "", readPhaseDelete)
	if err != nil {
		return "", err
	}
	currentID = responseID(response)
	if currentID == "" {
		return "", fmt.Errorf("%s delete confirmation could not resolve a resource OCID", c.config.Kind)
	}
	_ = mergeResponseIntoStatus(resource, response)
	return currentID, nil
}

func (c ServiceClient[T]) resolveExistingBeforeCreate(ctx context.Context, resource T) (any, error) {
	if skipExistingBeforeCreate(ctx) {
		return nil, nil
	}
	if !c.shouldResolveExistingBeforeCreate() {
		return nil, nil
	}

	response, err := c.readResource(ctx, resource, "", readPhaseCreate)
	if err == nil {
		return response, nil
	}
	if errors.Is(err, errResourceNotFound) {
		return nil, nil
	}
	return nil, err
}

func skipExistingBeforeCreate(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	skip, _ := ctx.Value(skipExistingBeforeCreateContextKey).(bool)
	return skip
}

func (c ServiceClient[T]) shouldResolveExistingBeforeCreate() bool {
	return c.config.Create != nil && c.config.List != nil && c.config.Semantics != nil && c.config.Semantics.List != nil
}

func (c ServiceClient[T]) requiresLiveMutationAssessment() bool {
	return c.config.Semantics != nil &&
		(len(c.config.Semantics.Mutation.ForceNew) > 0 || len(c.config.Semantics.Mutation.Mutable) > 0) &&
		(c.config.Get != nil || c.config.List != nil)
}

func (c ServiceClient[T]) shouldInvokeUpdate(resource T, currentResponse any) (bool, error) {
	if c.config.Update == nil {
		return false, nil
	}
	if c.shouldObserveCurrentLifecycle(currentResponse) {
		return false, nil
	}
	if c.config.Semantics == nil {
		return true, nil
	}
	return c.hasMutableDrift(resource, currentResponse)
}

func (c ServiceClient[T]) shouldObserveCurrentLifecycle(currentResponse any) bool {
	if c.config.Semantics == nil || currentResponse == nil {
		return false
	}

	lifecycleState := strings.ToUpper(responseLifecycleState(currentResponse))
	if lifecycleState == "" {
		return false
	}

	return containsString(c.config.Semantics.Lifecycle.ProvisioningStates, lifecycleState) ||
		containsString(c.config.Semantics.Lifecycle.UpdatingStates, lifecycleState) ||
		containsString(c.config.Semantics.Delete.PendingStates, lifecycleState)
}

func (c ServiceClient[T]) validateMutationPolicy(resource T, existing bool, currentResponse any) error {
	semantics := c.config.Semantics
	if semantics == nil {
		return nil
	}

	specValues, currentValues, err := mutationValues(resource, currentResponse)
	if err != nil {
		return err
	}
	if err := c.validateMutationConflicts(specValues); err != nil {
		return err
	}

	if !existing {
		return nil
	}
	if err := c.validateForceNewFields(specValues, currentValues); err != nil {
		return err
	}
	if c.config.Update == nil {
		return nil
	}

	unsupportedPaths := unsupportedUpdateDriftPaths(specValues, currentValues, semantics.Mutation)
	if len(unsupportedPaths) == 0 {
		return nil
	}
	return fmt.Errorf("%s formal semantics reject unsupported update drift for %s", c.config.Kind, strings.Join(unsupportedPaths, ", "))
}

func mutationValues(resource any, currentResponse any) (map[string]any, map[string]any, error) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil, nil, err
	}

	specValues := jsonMap(fieldInterface(resourceValue, "Spec"))
	currentValues := jsonMap(fieldInterface(resourceValue, "Status"))
	if body, ok := responseBody(currentResponse); ok && body != nil {
		mergeJSONMapOverwrite(currentValues, body)
	}
	return specValues, currentValues, nil
}

func (c ServiceClient[T]) validateMutationConflicts(specValues map[string]any) error {
	for field, conflicts := range c.config.Semantics.Mutation.ConflictsWith {
		if _, ok := lookupMeaningfulValue(specValues, field); !ok {
			continue
		}
		for _, conflict := range conflicts {
			if _, ok := lookupMeaningfulValue(specValues, conflict); ok {
				return fmt.Errorf("%s formal semantics forbid setting %s with %s", c.config.Kind, field, conflict)
			}
		}
	}
	return nil
}

func (c ServiceClient[T]) validateForceNewFields(specValues map[string]any, currentValues map[string]any) error {
	for _, field := range c.config.Semantics.Mutation.ForceNew {
		specValue, specOK := lookupMeaningfulValue(specValues, field)
		statusValue, statusOK := lookupValueByPath(currentValues, field)
		if !specOK || !statusOK {
			continue
		}
		if !forceNewValuesEqual(specValue, statusValue) {
			return fmt.Errorf("%s formal semantics require replacement when %s changes", c.config.Kind, field)
		}
	}
	return nil
}

func forceNewValuesEqual(specValue any, currentValue any) bool {
	specValue, specMeaningful := pruneComparableValue(specValue)
	currentValue, currentMeaningful := pruneComparableValue(currentValue)
	if !specMeaningful || !currentMeaningful {
		return !specMeaningful && !currentMeaningful
	}

	specMap, specIsMap := specValue.(map[string]any)
	currentMap, currentIsMap := currentValue.(map[string]any)
	if specIsMap && currentIsMap {
		return len(comparableDiffPaths(specMap, currentMap, "")) == 0
	}
	return valuesEqual(specValue, currentValue)
}

func pruneComparableValue(value any) (any, bool) {
	switch concrete := value.(type) {
	case map[string]any:
		pruned := make(map[string]any, len(concrete))
		for key, child := range concrete {
			prunedChild, ok := pruneComparableValue(child)
			if !ok {
				continue
			}
			pruned[key] = prunedChild
		}
		if len(pruned) == 0 {
			return nil, false
		}
		return pruned, true
	default:
		if !meaningfulValue(concrete) {
			return nil, false
		}
		return concrete, true
	}
}

func (c ServiceClient[T]) hasMutableDrift(resource T, currentResponse any) (bool, error) {
	semantics := c.config.Semantics
	if semantics == nil || len(semantics.Mutation.Mutable) == 0 {
		return false, nil
	}

	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return false, err
	}

	specValues := jsonMap(fieldInterface(resourceValue, "Spec"))
	currentValues := jsonMap(fieldInterface(resourceValue, "Status"))
	if body, ok := responseBody(currentResponse); ok && body != nil {
		currentValues = jsonMap(body)
	}

	for _, field := range semantics.Mutation.Mutable {
		specValue, specFound := lookupValueByPath(specValues, field)
		if !specFound || !meaningfulValue(specValue) {
			continue
		}
		currentValue, currentFound := lookupValueByPath(currentValues, field)
		if !currentFound || !meaningfulValue(currentValue) {
			if responseExposesFieldPath(currentResponse, field) {
				return true, nil
			}
			continue
		}
		if !valuesEqual(specValue, currentValue) {
			return true, nil
		}
	}

	return false, nil
}

func unsupportedUpdateDriftPaths(specValues map[string]any, currentValues map[string]any, semantics MutationSemantics) []string {
	diffPaths := comparableDiffPaths(specValues, currentValues, "")
	unsupported := make([]string, 0, len(diffPaths))
	for _, path := range diffPaths {
		switch {
		case pathCoveredByAny(path, semantics.Mutable):
		case pathCoveredByAny(path, semantics.ForceNew):
		default:
			unsupported = appendUniqueStrings(unsupported, path)
		}
	}
	sort.Strings(unsupported)
	return unsupported
}

func comparableDiffPaths(specValues map[string]any, currentValues map[string]any, prefix string) []string {
	if specValues == nil || currentValues == nil {
		return nil
	}

	keys := meaningfulSortedKeys(specValues)
	paths := make([]string, 0, len(keys))
	for _, key := range keys {
		paths = append(paths, comparableDiffPathsForKey(specValues, currentValues, prefix, key)...)
	}

	return paths
}

func meaningfulSortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if _, ok := pruneComparableValue(value); ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func comparableDiffPathsForKey(specValues map[string]any, currentValues map[string]any, prefix string, key string) []string {
	specValue := specValues[key]
	currentValue, ok := lookupMapKey(currentValues, key)
	if !ok {
		return nil
	}

	path := key
	if prefix != "" {
		path = prefix + "." + key
	}

	specMap, specIsMap := specValue.(map[string]any)
	currentMap, currentIsMap := currentValue.(map[string]any)
	if specIsMap && currentIsMap {
		return comparableDiffPaths(specMap, currentMap, path)
	}
	if !valuesEqual(specValue, currentValue) {
		return []string{path}
	}
	return nil
}

func pathCoveredByAny(path string, semanticPaths []string) bool {
	for _, semanticPath := range semanticPaths {
		if pathCoveredBy(path, semanticPath) {
			return true
		}
	}
	return false
}

func pathCoveredBy(path string, semanticPath string) bool {
	path = normalizePath(path)
	semanticPath = normalizePath(semanticPath)
	if path == "" || semanticPath == "" {
		return false
	}
	return path == semanticPath ||
		strings.HasPrefix(path, semanticPath+".") ||
		strings.HasPrefix(semanticPath, path+".")
}

func normalizePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}

	segments := strings.Split(path, ".")
	for index, segment := range segments {
		segments[index] = normalizePathSegment(segment)
	}
	return strings.Join(segments, ".")
}

func normalizePathSegment(segment string) string {
	segment = strings.ToLower(strings.TrimSpace(segment))
	if strings.HasSuffix(segment, "gbs") {
		return strings.TrimSuffix(segment, "gbs") + "gb"
	}
	return segment
}

func responseExposesFieldPath(response any, path string) bool {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return false
	}
	return typeExposesFieldPath(reflect.TypeOf(body), strings.Split(strings.TrimSpace(path), "."))
}

func typeExposesFieldPath(t reflect.Type, segments []string) bool {
	t = indirectType(t)
	if t == nil || len(segments) == 0 {
		return false
	}

	segment := strings.TrimSpace(segments[0])
	if segment == "" {
		return false
	}

	switch t.Kind() {
	case reflect.Struct:
		fieldType, ok := structFieldTypeByPathSegment(t, segment)
		if !ok {
			return false
		}
		if len(segments) == 1 {
			return true
		}
		return typeExposesFieldPath(fieldType, segments[1:])
	case reflect.Map:
		if len(segments) == 1 {
			return true
		}
		return typeExposesFieldPath(t.Elem(), segments[1:])
	case reflect.Slice, reflect.Array:
		return typeExposesFieldPath(t.Elem(), segments)
	default:
		return len(segments) == 1
	}
}

func indirectType(t reflect.Type) reflect.Type {
	for t != nil && (t.Kind() == reflect.Pointer || t.Kind() == reflect.Interface) {
		t = t.Elem()
	}
	return t
}

func structFieldTypeByPathSegment(t reflect.Type, segment string) (reflect.Type, bool) {
	normalized := normalizePathSegment(segment)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		if normalizePathSegment(field.Name) == normalized || normalizePathSegment(fieldJSONName(field)) == normalized {
			return field.Type, true
		}
	}
	return nil, false
}

func (c ServiceClient[T]) readResource(ctx context.Context, resource T, preferredID string, phase readPhase) (any, error) {
	state, err := c.prepareReadResourceState(resource, preferredID)
	if err != nil {
		return nil, err
	}

	state, response, handled, err := c.readResourceWithGet(ctx, resource, state, phase)
	if handled {
		return response, err
	}
	return c.readResourceWithList(ctx, resource, state, phase)
}

func (c ServiceClient[T]) readResourceForMutationValidation(ctx context.Context, resource T, currentID string, forceLiveGet bool) (any, error) {
	if !forceLiveGet {
		return c.readResource(ctx, resource, currentID, readPhaseUpdate)
	}
	if c.config.Get == nil {
		return nil, fmt.Errorf("%s generated runtime has no OCI Get operation for live mutation validation", c.config.Kind)
	}

	response, err := c.invoke(ctx, c.config.Get, resource, currentID, requestBuildOptions{})
	if err != nil {
		if isNotFound(err) {
			return nil, errResourceNotFound
		}
		return nil, err
	}
	return response, nil
}

func (c ServiceClient[T]) prepareReadResourceState(resource T, preferredID string) (readResourceState, error) {
	readID := preferredID
	if readID == "" {
		readID = c.currentID(resource)
	}

	values, err := lookupValues(resource)
	if err != nil {
		return readResourceState{}, err
	}

	return readResourceState{
		values:     values,
		readID:     readID,
		listValues: values,
		listID:     readID,
	}, nil
}

func (c ServiceClient[T]) readResourceWithGet(ctx context.Context, resource T, state readResourceState, phase readPhase) (readResourceState, any, bool, error) {
	if c.config.Get == nil || !c.canInvokeGet(resource, state.readID) {
		return state, nil, false, nil
	}

	response, err := c.invoke(ctx, c.config.Get, resource, state.readID, requestBuildOptions{})
	if err == nil {
		return state, response, true, nil
	}
	if !isNotFound(err) || c.config.List == nil {
		return state, nil, true, err
	}

	return c.fallbackReadResourceState(resource, state, phase), nil, false, nil
}

func (c ServiceClient[T]) fallbackReadResourceState(resource T, state readResourceState, phase readPhase) readResourceState {
	if phase != readPhaseDelete && c.usesStatusOnlyCurrentID(resource, state.readID) {
		state.listValues = valuesWithoutAliases(state.values, c.idFieldAliases())
		state.listID = ""
	}
	return state
}

func (c ServiceClient[T]) readResourceWithList(ctx context.Context, resource T, state readResourceState, phase readPhase) (any, error) {
	if c.config.List == nil {
		return nil, fmt.Errorf("%s generated runtime has no readable OCI operation", c.config.Kind)
	}

	response, err := c.invokeWithValues(ctx, c.config.List, resource, state.listValues, state.listID, requestBuildOptions{})
	if err != nil {
		return nil, err
	}

	body, ok := responseBody(response)
	if !ok {
		return nil, fmt.Errorf("%s list response did not expose a body payload", c.config.Kind)
	}
	return c.selectListItem(body, state.listValues, state.listID, phase)
}

func (c ServiceClient[T]) canInvokeExplicitGet(values map[string]any, preferredID string) bool {
	for _, field := range c.config.Get.Fields {
		if !requestFieldRequiresResourceID(field, c.idFieldAliases()) {
			continue
		}
		if _, ok := explicitRequestValue(values, field, preferredID); !ok {
			return false
		}
	}
	return true
}

func (c ServiceClient[T]) canInvokeHeuristicGet(values map[string]any, preferredID string) bool {
	requestStruct, ok := operationRequestStruct(c.config.Get.NewRequest)
	if !ok {
		return true
	}

	for i := 0; i < requestStruct.NumField(); i++ {
		fieldType, inspect := heuristicGetField(requestStruct, i)
		if !inspect {
			continue
		}
		if !c.canPopulateHeuristicGetField(values, preferredID, fieldType) {
			return false
		}
	}

	return true
}

func heuristicGetField(requestStruct reflect.Value, index int) (reflect.StructField, bool) {
	fieldValue := requestStruct.Field(index)
	fieldType := requestStruct.Type().Field(index)
	if !fieldValue.CanSet() || fieldType.Name == "RequestMetadata" {
		return reflect.StructField{}, false
	}

	switch fieldType.Tag.Get("contributesTo") {
	case "header", "binary", "body":
		return reflect.StructField{}, false
	default:
		return fieldType, true
	}
}

func (c ServiceClient[T]) canPopulateHeuristicGetField(values map[string]any, preferredID string, fieldType reflect.StructField) bool {
	lookupKey := requestLookupKey(fieldType)
	if !containsString(c.idFieldAliases(), lookupKey) || preferredID != "" {
		return true
	}
	_, ok := lookupValueByPaths(values, lookupKey)
	return ok
}

func (c ServiceClient[T]) canInvokeGet(resource T, preferredID string) bool {
	if c.config.Get == nil {
		return false
	}

	values, err := lookupValues(resource)
	if err != nil {
		return true
	}

	if len(c.config.Get.Fields) > 0 {
		return c.canInvokeExplicitGet(values, preferredID)
	}
	return c.canInvokeHeuristicGet(values, preferredID)
}

func (c ServiceClient[T]) invoke(ctx context.Context, op *Operation, resource T, preferredID string, options requestBuildOptions) (any, error) {
	values, err := lookupValues(resource)
	if err != nil {
		return nil, err
	}
	return c.invokeWithValues(ctx, op, resource, values, preferredID, options)
}

func (c ServiceClient[T]) invokeWithValues(ctx context.Context, op *Operation, resource T, values map[string]any, preferredID string, options requestBuildOptions) (any, error) {
	if op == nil {
		return nil, fmt.Errorf("%s generated runtime does not define this OCI operation", c.config.Kind)
	}
	if op.NewRequest == nil || op.Call == nil {
		return nil, fmt.Errorf("%s generated runtime OCI operation is incomplete", c.config.Kind)
	}

	request := op.NewRequest()
	if request == nil {
		return nil, fmt.Errorf("%s generated runtime did not create an OCI request value", c.config.Kind)
	}
	bodyOverride, hasBodyOverride, err := c.requestBodyOverride(op, resource, options)
	if err != nil {
		return nil, err
	}
	if err := buildRequest(request, resource, values, preferredID, op.Fields, c.idFieldAliases(), options, bodyOverride, hasBodyOverride); err != nil {
		return nil, fmt.Errorf("build %s OCI request: %w", c.config.Kind, err)
	}

	response, err := op.Call(ctx, request)
	if err != nil {
		return nil, normalizeOCIError(err)
	}
	return response, nil
}

func (c ServiceClient[T]) requestBodyOverride(op *Operation, resource T, options requestBuildOptions) (any, bool, error) {
	if op == c.config.Create && c.config.BuildCreateBody != nil {
		body, err := c.config.BuildCreateBody(options.Context, resource, options.Namespace)
		if err != nil {
			return nil, false, fmt.Errorf("build %s create body: %w", c.config.Kind, err)
		}
		return body, true, nil
	}
	if op == c.config.Update {
		body, ok, err := c.filteredUpdateBody(resource, options)
		if err != nil {
			return nil, false, fmt.Errorf("build %s update body: %w", c.config.Kind, err)
		}
		if ok {
			return body, true, nil
		}
	}
	return nil, false, nil
}

func (c ServiceClient[T]) filteredUpdateBody(resource T, options requestBuildOptions) (any, bool, error) {
	if c.config.Update == nil || c.config.Semantics == nil || len(c.config.Semantics.Mutation.Mutable) == 0 {
		return nil, false, nil
	}

	resolvedSpec, err := resolvedSpecValue(resource, options)
	if err != nil {
		return nil, false, err
	}
	specValues := jsonMap(resolvedSpec)
	if len(specValues) == 0 {
		return nil, false, nil
	}

	currentValues := map[string]any{}
	if options.CurrentResponse != nil {
		if body, ok := responseBody(options.CurrentResponse); ok && body != nil {
			currentValues = jsonMap(body)
		}
	}
	if len(currentValues) == 0 {
		statusValue, err := statusStruct(resource)
		if err != nil {
			return nil, false, err
		}
		currentValues = jsonMap(statusValue.Interface())
	}

	body := make(map[string]any)
	for _, path := range c.config.Semantics.Mutation.Mutable {
		specValue, ok := lookupMeaningfulValue(specValues, path)
		if !ok {
			continue
		}
		if currentValue, currentFound := lookupValueByPath(currentValues, path); currentFound && valuesEqual(specValue, currentValue) {
			continue
		}
		setValueByPath(body, canonicalValuePath(specValues, path), specValue)
	}
	if len(body) == 0 {
		return nil, false, nil
	}
	return body, true, nil
}

func (c ServiceClient[T]) applySuccess(resource T, response any, fallback shared.OSOKConditionType) (servicemanager.OSOKResponse, error) {
	if err := mergeResponseIntoStatus(resource, response); err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}
	stampSecretSourceStatus(resource)

	status, err := osokStatus(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}, err
	}

	resourceID := responseID(response)
	if resourceID == "" {
		resourceID = c.currentID(resource)
	}
	if resourceID != "" {
		status.Ocid = shared.OCID(resourceID)
	}

	conditionType, shouldRequeue, message := c.classifyLifecycle(response, fallback)
	status.Message = message
	status.Reason = string(conditionType)
	now := metav1.Now()
	if resourceID != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, conditionType, v1.ConditionTrue, "", message, c.config.Log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    conditionType != shared.Failed,
		ShouldRequeue:   shouldRequeue,
		RequeueDuration: defaultRequeueDuration,
	}, nil
}

func (c ServiceClient[T]) markFailure(resource T, err error) error {
	status, statusErr := osokStatus(resource)
	if statusErr != nil {
		return err
	}
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, shared.Failed, v1.ConditionFalse, "", err.Error(), c.config.Log)
	return err
}

func (c ServiceClient[T]) markDeleted(resource T, message string) {
	status, err := osokStatus(resource)
	if err != nil {
		return
	}
	now := metav1.Now()
	status.DeletedAt = &now
	status.UpdatedAt = &now
	if message != "" {
		status.Message = message
	}
	status.Reason = string(shared.Terminating)
	*status = util.UpdateOSOKStatusCondition(*status, shared.Terminating, v1.ConditionTrue, "", status.Message, c.config.Log)
}

func (c ServiceClient[T]) markCondition(resource T, condition shared.OSOKConditionType, message string) {
	status, err := osokStatus(resource)
	if err != nil {
		return
	}
	now := metav1.Now()
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(condition)
	*status = util.UpdateOSOKStatusCondition(*status, condition, v1.ConditionTrue, "", message, c.config.Log)
}

func (c ServiceClient[T]) currentID(resource T) string {
	return c.statusID(resource)
}

func (c ServiceClient[T]) usesStatusOnlyCurrentID(resource T, currentID string) bool {
	if currentID == "" {
		return false
	}
	return currentID == c.statusID(resource) && c.specID(resource) == ""
}

func (c ServiceClient[T]) statusID(resource T) string {
	status, err := osokStatus(resource)
	if err == nil && status.Ocid != "" {
		return string(status.Ocid)
	}

	statusValue, err := statusStruct(resource)
	if err != nil {
		return ""
	}
	return firstNonEmpty(jsonMap(statusValue.Interface()), c.idFieldAliases()...)
}

func (c ServiceClient[T]) specID(resource T) string {
	return firstNonEmpty(jsonMap(specValue(resource)), c.idFieldAliases()...)
}

func (c ServiceClient[T]) idFieldAliases() []string {
	aliases := []string{"id", "ocid"}
	for _, name := range []string{c.config.Kind, c.config.SDKName} {
		if strings.TrimSpace(name) == "" {
			continue
		}
		aliases = appendUniqueStrings(aliases, lowerCamel(name)+"Id")
	}
	return aliases
}

type requestBuildOptions struct {
	Context          context.Context
	CredentialClient credhelper.CredentialClient
	Namespace        string
	CurrentResponse  any
}

func buildRequest(
	request any,
	resource any,
	values map[string]any,
	preferredID string,
	fields []RequestField,
	idAliases []string,
	options requestBuildOptions,
	bodyOverride any,
	hasBodyOverride bool,
) error {
	requestValue := reflect.ValueOf(request)
	if !requestValue.IsValid() || requestValue.Kind() != reflect.Pointer || requestValue.IsNil() {
		return fmt.Errorf("expected pointer OCI request, got %T", request)
	}

	requestStruct := requestValue.Elem()
	if requestStruct.Kind() != reflect.Struct {
		return fmt.Errorf("expected pointer to OCI request struct, got %T", request)
	}

	var resolvedSpec any
	switch {
	case hasBodyOverride:
		resolvedSpec = bodyOverride
	case requestNeedsResolvedSpec(fields, requestStruct.Type()):
		var err error
		resolvedSpec, err = resolvedSpecValue(resource, options)
		if err != nil {
			return err
		}
	}

	if len(fields) > 0 {
		if err := buildExplicitRequest(requestStruct, values, preferredID, fields, resolvedSpec); err != nil {
			return err
		}
		assignDeterministicRetryToken(requestStruct, resource)
		return nil
	}

	if err := buildHeuristicRequest(requestStruct, requestStruct.Type(), values, preferredID, idAliases, resolvedSpec); err != nil {
		return err
	}
	assignDeterministicRetryToken(requestStruct, resource)
	return nil
}

func buildExplicitRequest(requestStruct reflect.Value, values map[string]any, preferredID string, fields []RequestField, resolvedSpec any) error {
	for _, field := range fields {
		fieldValue := requestStruct.FieldByName(field.FieldName)
		if !fieldValue.IsValid() || !fieldValue.CanSet() {
			continue
		}

		switch field.Contribution {
		case "header", "binary":
			continue
		case "body":
			if err := assignField(fieldValue, resolvedSpec); err != nil {
				return fmt.Errorf("set body field %s: %w", field.FieldName, err)
			}
			continue
		}

		rawValue, ok := explicitRequestValue(values, field, preferredID)
		if !ok {
			continue
		}
		if err := assignField(fieldValue, rawValue); err != nil {
			return fmt.Errorf("set request field %s: %w", field.FieldName, err)
		}
	}

	return nil
}

// ResolveSpecValue rewrites secret-backed spec inputs and omits zero-value nested
// structs using the same projection rules as generated runtime request builders.
func ResolveSpecValue(resource any, ctx context.Context, credentialClient credhelper.CredentialClient, namespace string) (any, error) {
	return resolvedSpecValue(resource, requestBuildOptions{
		Context:          ctx,
		CredentialClient: credentialClient,
		Namespace:        namespace,
	})
}

func buildHeuristicRequest(
	requestStruct reflect.Value,
	requestType reflect.Type,
	values map[string]any,
	preferredID string,
	idAliases []string,
	resolvedSpec any,
) error {
	for i := 0; i < requestStruct.NumField(); i++ {
		if err := populateHeuristicRequestField(requestStruct.Field(i), requestType.Field(i), values, preferredID, idAliases, resolvedSpec); err != nil {
			return err
		}
	}

	return nil
}

func operationRequestStruct(newRequest func() any) (reflect.Value, bool) {
	if newRequest == nil {
		return reflect.Value{}, false
	}

	request := newRequest()
	if request == nil {
		return reflect.Value{}, false
	}

	requestValue := reflect.ValueOf(request)
	if !requestValue.IsValid() || requestValue.Kind() != reflect.Pointer || requestValue.IsNil() {
		return reflect.Value{}, false
	}

	requestStruct := requestValue.Elem()
	if requestStruct.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	return requestStruct, true
}

func requestLookupKey(fieldType reflect.StructField) string {
	lookupKey := fieldType.Tag.Get("name")
	if lookupKey == "" {
		lookupKey = fieldJSONName(fieldType)
	}
	if lookupKey == "" {
		lookupKey = lowerCamel(fieldType.Name)
	}
	return lookupKey
}

func populateHeuristicRequestField(fieldValue reflect.Value, fieldType reflect.StructField, values map[string]any, preferredID string, idAliases []string, resolvedSpec any) error {
	if !fieldValue.CanSet() || fieldType.Name == "RequestMetadata" {
		return nil
	}

	switch fieldType.Tag.Get("contributesTo") {
	case "header", "binary":
		return nil
	case "body":
		if err := assignField(fieldValue, resolvedSpec); err != nil {
			return fmt.Errorf("set body field %s: %w", fieldType.Name, err)
		}
		return nil
	}

	rawValue, ok := heuristicRequestValue(values, fieldType, preferredID, idAliases)
	if !ok {
		return nil
	}
	if err := assignField(fieldValue, rawValue); err != nil {
		return fmt.Errorf("set request field %s: %w", fieldType.Name, err)
	}
	return nil
}

func heuristicRequestValue(values map[string]any, fieldType reflect.StructField, preferredID string, idAliases []string) (any, bool) {
	lookupKey := requestLookupKey(fieldType)
	if lookupKey == "namespaceName" {
		if value, ok := lookupValueByPaths(values, "namespace"); ok {
			return value, true
		}
		if value, ok := lookupValueByPaths(values, "namespaceName"); ok {
			return value, true
		}
		return nil, false
	}
	if rawValue, ok := lookupValueByPaths(values, lookupKey); ok {
		return rawValue, true
	}
	if preferredID != "" && containsString(idAliases, lookupKey) {
		return preferredID, true
	}
	switch lookupKey {
	case "name":
		return lookupValueByPaths(values, "metadataName")
	default:
		return nil, false
	}
}

func explicitRequestValue(values map[string]any, field RequestField, preferredID string) (any, bool) {
	if field.PreferResourceID {
		if preferredID != "" {
			return preferredID, true
		}
		if currentID, ok := lookupValueByPaths(values, "id", "ocid"); ok {
			return currentID, true
		}
		if len(field.LookupPaths) != 0 {
			if rawValue, ok := lookupValueByPaths(values, field.LookupPaths...); ok {
				return rawValue, true
			}
		}
		return nil, false
	}

	lookupKey := strings.TrimSpace(field.RequestName)
	if lookupKey == "" {
		lookupKey = lowerCamel(field.FieldName)
	}

	if len(field.LookupPaths) != 0 {
		if rawValue, ok := lookupValueByPaths(values, field.LookupPaths...); ok {
			return rawValue, true
		}
	}
	if lookupKey == "namespaceName" {
		if value, ok := lookupValueByPaths(values, "namespace"); ok {
			return value, true
		}
		if value, ok := lookupValueByPaths(values, "namespaceName"); ok {
			return value, true
		}
		return nil, false
	}

	if rawValue, ok := lookupValueByPaths(values, lookupKey); ok {
		return rawValue, true
	}
	if lookupKey == "name" {
		return lookupValueByPaths(values, "metadataName")
	}

	return nil, false
}

func setValueByPath(values map[string]any, path string, value any) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}

	segments := strings.Split(path, ".")
	current := values
	for index, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return
		}
		if index == len(segments)-1 {
			current[segment] = value
			return
		}
		next, ok := current[segment].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[segment] = next
		}
		current = next
	}
}

func canonicalValuePath(values map[string]any, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	segments := strings.Split(path, ".")
	resolved := make([]string, 0, len(segments))
	current := values
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return strings.Join(resolved, ".")
		}
		key := canonicalMapKey(current, segment)
		resolved = append(resolved, key)

		next, ok := current[key].(map[string]any)
		if !ok {
			current = nil
			continue
		}
		current = next
	}
	return strings.Join(resolved, ".")
}

func canonicalMapKey(values map[string]any, segment string) string {
	if values == nil {
		return segment
	}
	normalized := normalizePathSegment(segment)
	for key := range values {
		if normalizePathSegment(key) == normalized {
			return key
		}
	}
	return segment
}

func requestFieldRequiresResourceID(field RequestField, idAliases []string) bool {
	if field.PreferResourceID {
		return true
	}

	lookupKey := strings.TrimSpace(field.RequestName)
	if lookupKey == "" {
		lookupKey = lowerCamel(field.FieldName)
	}
	return containsString(idAliases, lookupKey)
}

func lookupValues(resource any) (map[string]any, error) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil, err
	}

	values := make(map[string]any)
	mergeJSONMap(values, fieldInterface(resourceValue, "Spec"))
	mergeJSONMap(values, fieldInterface(resourceValue, "Status"))
	if statusField, ok := fieldValue(resourceValue, "Status"); ok {
		mergeJSONMap(values, fieldInterface(statusField, "OsokStatus"))
	}

	if metadataName := lookupMetadataString(resourceValue, "Name"); metadataName != "" {
		if _, exists := values["name"]; !exists {
			values["name"] = metadataName
		}
		values["metadataName"] = metadataName
	}
	if namespaceName := lookupMetadataString(resourceValue, "Namespace"); namespaceName != "" {
		if _, exists := values["namespaceName"]; !exists {
			values["namespaceName"] = namespaceName
		}
		if _, exists := values["namespace"]; !exists {
			values["namespace"] = namespaceName
		}
	}

	return values, nil
}

func mergeJSONMap(dst map[string]any, source any) {
	if source == nil {
		return
	}
	payload, err := json.Marshal(source)
	if err != nil {
		return
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return
	}
	for key, value := range decoded {
		if _, exists := dst[key]; exists {
			continue
		}
		dst[key] = value
	}
}

func mergeJSONMapOverwrite(dst map[string]any, source any) {
	if source == nil {
		return
	}
	payload, err := json.Marshal(source)
	if err != nil {
		return
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return
	}
	for key, value := range decoded {
		dst[key] = value
	}
}

func specValue(resource any) any {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil
	}
	return fieldInterface(resourceValue, "Spec")
}

func requestNeedsResolvedSpec(fields []RequestField, requestType reflect.Type) bool {
	if len(fields) > 0 {
		for _, field := range fields {
			if field.Contribution == "body" {
				return true
			}
		}
		return false
	}

	for i := 0; i < requestType.NumField(); i++ {
		if requestType.Field(i).Tag.Get("contributesTo") == "body" {
			return true
		}
	}
	return false
}

func resolvedSpecValue(resource any, options requestBuildOptions) (any, error) {
	raw := specValue(resource)
	if raw == nil {
		return nil, nil
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal spec value: %w", err)
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("decode spec value: %w", err)
	}

	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil, err
	}
	specField, ok := fieldValue(resourceValue, "Spec")
	if !ok {
		return nil, fmt.Errorf("resource %T does not expose Spec", resource)
	}

	resolved, _, err := rewriteSecretSources(specField, decoded, options)
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

func indirectValue(value reflect.Value) (reflect.Value, bool) {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
	return value, value.IsValid()
}

func rewriteSecretSources(value reflect.Value, decoded any, options requestBuildOptions) (any, bool, error) {
	value, ok := indirectValue(value)
	if !ok {
		return nil, false, nil
	}
	if rewritten, include, handled, err := rewriteSharedSecretSource(value, options); handled {
		return rewritten, include, err
	}
	if value.Kind() == reflect.Struct && value.IsZero() {
		return nil, false, nil
	}

	switch value.Kind() {
	case reflect.Struct:
		return rewriteSecretStruct(value, decoded, options)
	case reflect.Slice, reflect.Array:
		return rewriteSecretSlice(value, decoded, options)
	case reflect.Map:
		return rewriteSecretMap(value, decoded, options)
	default:
		return decoded, true, nil
	}
}

func rewriteSharedSecretSource(value reflect.Value, options requestBuildOptions) (any, bool, bool, error) {
	switch value.Type() {
	case passwordSourceType:
		rewritten, include, err := resolveSecretSourceValue(options.Context, options.CredentialClient, options.Namespace, value.FieldByName("Secret"), "SecretName", "password")
		return rewritten, include, true, err
	case usernameSourceType:
		rewritten, include, err := resolveSecretSourceValue(options.Context, options.CredentialClient, options.Namespace, value.FieldByName("Secret"), "SecretName", "username")
		return rewritten, include, true, err
	default:
		return nil, false, false, nil
	}
}

func rewriteSecretStruct(value reflect.Value, decoded any, options requestBuildOptions) (any, bool, error) {
	decodedMap := decodedMapValue(decoded)
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() {
			continue
		}
		if fieldType.Anonymous && embeddedJSONField(fieldType) {
			var err error
			decodedMap, err = rewriteEmbeddedSecretField(value.Field(i), decodedMap, options)
			if err != nil {
				return nil, false, err
			}
			continue
		}
		if err := rewriteNamedSecretField(decodedMap, value.Field(i), fieldType, options); err != nil {
			return nil, false, err
		}
	}
	return decodedMap, true, nil
}

func decodedMapValue(decoded any) map[string]any {
	decodedMap, ok := decoded.(map[string]any)
	if !ok || decodedMap == nil {
		return map[string]any{}
	}
	return decodedMap
}

func rewriteEmbeddedSecretField(fieldValue reflect.Value, decodedMap map[string]any, options requestBuildOptions) (map[string]any, error) {
	rewritten, _, err := rewriteSecretSources(fieldValue, decodedMap, options)
	if err != nil {
		return nil, err
	}
	if nestedMap, ok := rewritten.(map[string]any); ok {
		return nestedMap, nil
	}
	return decodedMap, nil
}

func rewriteNamedSecretField(decodedMap map[string]any, fieldValue reflect.Value, fieldType reflect.StructField, options requestBuildOptions) error {
	jsonName, skip := fieldJSONTagName(fieldType)
	if skip {
		return nil
	}
	childDecoded, exists := decodedMap[jsonName]
	rewritten, include, err := rewriteSecretSources(fieldValue, childDecoded, options)
	if err != nil {
		return err
	}
	if include {
		decodedMap[jsonName] = rewritten
		return nil
	}
	if exists {
		delete(decodedMap, jsonName)
	}
	return nil
}

func rewriteSecretSlice(value reflect.Value, decoded any, options requestBuildOptions) (any, bool, error) {
	decodedSlice, ok := decoded.([]any)
	if !ok {
		return decoded, true, nil
	}
	for i := 0; i < value.Len() && i < len(decodedSlice); i++ {
		rewritten, include, err := rewriteSecretSources(value.Index(i), decodedSlice[i], options)
		if err != nil {
			return nil, false, err
		}
		if include {
			decodedSlice[i] = rewritten
		}
	}
	return decodedSlice, true, nil
}

func rewriteSecretMap(value reflect.Value, decoded any, options requestBuildOptions) (any, bool, error) {
	if value.Type().Key().Kind() != reflect.String {
		return decoded, true, nil
	}
	decodedMap, ok := decoded.(map[string]any)
	if !ok {
		return decoded, true, nil
	}
	iter := value.MapRange()
	for iter.Next() {
		key := iter.Key().String()
		childDecoded, exists := decodedMap[key]
		rewritten, include, err := rewriteSecretSources(iter.Value(), childDecoded, options)
		if err != nil {
			return nil, false, err
		}
		if include {
			decodedMap[key] = rewritten
			continue
		}
		if exists {
			delete(decodedMap, key)
		}
	}
	return decodedMap, true, nil
}

func resolveSecretSourceValue(
	ctx context.Context,
	credentialClient credhelper.CredentialClient,
	namespace string,
	secretField reflect.Value,
	nameField string,
	dataKey string,
) (any, bool, error) {
	if !secretField.IsValid() {
		return nil, false, nil
	}
	secretNameField := secretField.FieldByName(nameField)
	if !secretNameField.IsValid() || secretNameField.Kind() != reflect.String {
		return nil, false, nil
	}

	secretName := strings.TrimSpace(secretNameField.String())
	if secretName == "" {
		return nil, false, nil
	}
	if credentialClient == nil {
		return nil, false, fmt.Errorf("resolve %s secret %q: credential client is nil", dataKey, secretName)
	}
	if strings.TrimSpace(namespace) == "" {
		return nil, false, fmt.Errorf("resolve %s secret %q: namespace is empty", dataKey, secretName)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	secretData, err := credentialClient.GetSecret(ctx, secretName, namespace)
	if err != nil {
		return nil, false, fmt.Errorf("get %s secret %q: %w", dataKey, secretName, err)
	}
	rawValue, ok := secretData[dataKey]
	if !ok {
		return nil, false, fmt.Errorf("%s key in secret %q is not found", dataKey, secretName)
	}
	return string(rawValue), true, nil
}

func fieldJSONTagName(field reflect.StructField) (string, bool) {
	name := field.Tag.Get("json")
	if name == "-" {
		return "", true
	}
	if strings.TrimSpace(name) == "" {
		return lowerCamel(field.Name), false
	}
	parts := strings.Split(name, ",")
	if len(parts) == 0 || parts[0] == "" {
		return lowerCamel(field.Name), false
	}
	return parts[0], false
}

func embeddedJSONField(field reflect.StructField) bool {
	if !field.Anonymous {
		return false
	}
	parts := strings.Split(field.Tag.Get("json"), ",")
	return len(parts) == 0 || parts[0] == ""
}

func responseBody(response any) (any, bool) {
	if response == nil {
		return nil, false
	}

	value, ok := indirectValue(reflect.ValueOf(response))
	if !ok {
		return nil, false
	}
	if value.Kind() != reflect.Struct {
		return response, true
	}

	if !strings.HasSuffix(value.Type().Name(), "Response") {
		return value.Interface(), true
	}
	return responseStructBody(value)
}

func responseStructBody(value reflect.Value) (any, bool) {
	typ := value.Type()
	var fallback reflect.Value
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		fieldValue := value.Field(i)

		body, ok := taggedResponseBody(fieldType, fieldValue)
		if ok {
			return body, body != nil
		}
		if shouldSkipResponseFallback(fieldType) || fallback.IsValid() {
			continue
		}
		fallback = fieldValue
	}

	if fallback.IsValid() {
		return fallback.Interface(), true
	}
	return nil, false
}

func taggedResponseBody(fieldType reflect.StructField, fieldValue reflect.Value) (any, bool) {
	if !fieldType.IsExported() || fieldType.Tag.Get("presentIn") != "body" {
		return nil, false
	}
	if fieldValue.Kind() != reflect.Pointer {
		return fieldValue.Interface(), true
	}
	if fieldValue.IsNil() {
		return nil, true
	}
	return fieldValue.Interface(), true
}

func shouldSkipResponseFallback(fieldType reflect.StructField) bool {
	if !fieldType.IsExported() {
		return true
	}
	return fieldType.Name == "RawResponse" || strings.HasPrefix(fieldType.Name, "Opc") || fieldType.Name == "Etag"
}

func mergeResponseIntoStatus(resource any, response any) error {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return nil
	}

	statusValue, err := statusStruct(resource)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal OCI response body: %w", err)
	}
	if err := json.Unmarshal(payload, statusValue.Addr().Interface()); err != nil {
		return fmt.Errorf("project OCI response body into status: %w", err)
	}
	return nil
}

func stampSecretSourceStatus(resource any) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return
	}

	specField, ok := fieldValue(resourceValue, "Spec")
	if !ok {
		return
	}
	statusField, ok := fieldValue(resourceValue, "Status")
	if !ok {
		return
	}

	copySecretSourceFields(specField, statusField)
}

func copySecretSourceFields(source reflect.Value, destination reflect.Value) {
	source, destination, ok := secretSourceStructPair(source, destination)
	if !ok {
		return
	}
	for i := 0; i < source.NumField(); i++ {
		copySecretSourceField(source, destination, i)
	}
}

func secretSourceStructPair(source reflect.Value, destination reflect.Value) (reflect.Value, reflect.Value, bool) {
	var ok bool
	source, ok = indirectValue(source)
	if !ok || source.Kind() != reflect.Struct {
		return reflect.Value{}, reflect.Value{}, false
	}
	destination, ok = indirectValue(destination)
	if !ok || destination.Kind() != reflect.Struct {
		return reflect.Value{}, reflect.Value{}, false
	}
	return source, destination, true
}

func copySecretSourceField(source reflect.Value, destination reflect.Value, index int) {
	fieldType := source.Type().Field(index)
	if !fieldType.IsExported() {
		return
	}

	sourceField := source.Field(index)
	destinationField, ok := settableFieldByName(destination, fieldType.Name)
	if !ok {
		return
	}
	if copySecretSourceLeaf(sourceField, destinationField) {
		return
	}
	copySecretSourceFields(sourceField, destinationField)
}

func settableFieldByName(value reflect.Value, name string) (reflect.Value, bool) {
	field := value.FieldByName(name)
	if !field.IsValid() || !field.CanSet() {
		return reflect.Value{}, false
	}
	return field, true
}

func copySecretSourceLeaf(source reflect.Value, destination reflect.Value) bool {
	if !isSecretSourceType(source.Type()) || destination.Type() != source.Type() {
		return false
	}
	if secretSourceValueIsEmpty(source) {
		destination.Set(reflect.Zero(destination.Type()))
		return true
	}
	destination.Set(source)
	return true
}

func secretSourceValueIsEmpty(value reflect.Value) bool {
	var ok bool
	value, ok = indirectValue(value)
	if !ok || value.Kind() != reflect.Struct {
		return true
	}
	secretField := value.FieldByName("Secret")
	secretField, ok = indirectValue(secretField)
	if !ok || secretField.Kind() != reflect.Struct {
		return true
	}
	nameField := secretField.FieldByName("SecretName")
	if !nameField.IsValid() || nameField.Kind() != reflect.String {
		return true
	}
	return strings.TrimSpace(nameField.String()) == ""
}

func isSecretSourceType(typ reflect.Type) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ == usernameSourceType || typ == passwordSourceType
}

func responseID(response any) string {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return ""
	}
	values := jsonMap(body)
	return firstNonEmpty(values, "id", "ocid")
}

func (c ServiceClient[T]) classifyLifecycle(response any, fallback shared.OSOKConditionType) (shared.OSOKConditionType, bool, string) {
	if c.config.Semantics == nil {
		return classifyLifecycleHeuristics(response, fallback)
	}
	return classifyLifecycleSemantics(response, fallback, c.config.Semantics)
}

func classifyLifecycleSemantics(response any, fallback shared.OSOKConditionType, semantics *Semantics) (shared.OSOKConditionType, bool, string) {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return fallback, shouldRequeueForCondition(fallback), defaultConditionMessage(fallback)
	}

	values := jsonMap(body)
	lifecycleState := strings.ToUpper(firstNonEmpty(values, "lifecycleState", "status"))
	message := firstNonEmpty(values, "lifecycleDetails", "message", "displayName", "name")
	if message == "" {
		message = defaultConditionMessage(fallback)
	}

	switch {
	case lifecycleState == "":
		return fallback, shouldRequeueForCondition(fallback), message
	case containsString(semantics.Lifecycle.ProvisioningStates, lifecycleState):
		return shared.Provisioning, true, message
	case containsString(semantics.Lifecycle.UpdatingStates, lifecycleState):
		return shared.Updating, true, message
	case containsString(semantics.Delete.PendingStates, lifecycleState):
		return shared.Terminating, true, message
	case containsString(semantics.Lifecycle.ActiveStates, lifecycleState),
		containsString(semantics.Delete.TerminalStates, lifecycleState):
		return shared.Active, false, message
	default:
		return shared.Failed, false, fmt.Sprintf("formal lifecycle state %q is not modeled: %s", lifecycleState, message)
	}
}

func classifyLifecycleHeuristics(response any, fallback shared.OSOKConditionType) (shared.OSOKConditionType, bool, string) {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return fallback, shouldRequeueForCondition(fallback), defaultConditionMessage(fallback)
	}

	values := jsonMap(body)
	lifecycleState := strings.ToUpper(firstNonEmpty(values, "lifecycleState", "status"))
	message := firstNonEmpty(values, "lifecycleDetails", "message", "displayName", "name")
	if message == "" {
		message = defaultConditionMessage(fallback)
	}

	switch {
	case lifecycleState == "":
		return fallback, shouldRequeueForCondition(fallback), message
	case strings.Contains(lifecycleState, "FAIL"),
		strings.Contains(lifecycleState, "ERROR"),
		strings.Contains(lifecycleState, "NEEDS_ATTENTION"),
		strings.Contains(lifecycleState, "INOPERABLE"):
		return shared.Failed, false, message
	case strings.Contains(lifecycleState, "DELETE"),
		strings.Contains(lifecycleState, "TERMINAT"):
		return shared.Terminating, true, message
	case strings.Contains(lifecycleState, "UPDAT"),
		strings.Contains(lifecycleState, "MODIFY"),
		strings.Contains(lifecycleState, "PATCH"):
		return shared.Updating, true, message
	case strings.Contains(lifecycleState, "CREATE"),
		strings.Contains(lifecycleState, "PROVISION"),
		strings.Contains(lifecycleState, "PENDING"),
		strings.Contains(lifecycleState, "IN_PROGRESS"),
		strings.Contains(lifecycleState, "ACCEPT"),
		strings.Contains(lifecycleState, "START"):
		return shared.Provisioning, true, message
	default:
		return shared.Active, false, message
	}
}

func shouldRequeueForCondition(condition shared.OSOKConditionType) bool {
	return condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating
}

func defaultConditionMessage(condition shared.OSOKConditionType) string {
	switch condition {
	case shared.Provisioning:
		return "OCI resource provisioning is in progress"
	case shared.Updating:
		return "OCI resource update is in progress"
	case shared.Terminating:
		return "OCI resource delete is in progress"
	case shared.Failed:
		return "OCI resource reconcile failed"
	default:
		return "OCI resource is active"
	}
}

func validateFormalSemantics(kind string, semantics *Semantics) error {
	if semantics == nil {
		return nil
	}

	problems := append([]string{}, unsupportedFormalProblems(semantics)...)
	problems = append(problems, unsupportedAuxiliaryProblems(semantics)...)
	problems = append(problems, unsupportedFollowUpHelpers(semantics)...)
	problems = append(problems, invalidListSemanticsProblems(semantics)...)
	problems = append(problems, invalidDeleteSemanticsProblems(semantics)...)
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%s formal semantics blocked: %s", kind, strings.Join(problems, "; "))
}

func unsupportedFormalProblems(semantics *Semantics) []string {
	problems := make([]string, 0, len(semantics.Unsupported))
	for _, gap := range semantics.Unsupported {
		problems = append(problems, fmt.Sprintf("open formal gap %s: %s", gap.Category, gap.StopCondition))
	}
	return problems
}

func unsupportedAuxiliaryProblems(semantics *Semantics) []string {
	problems := make([]string, 0, len(semantics.AuxiliaryOperations))
	for _, operation := range semantics.AuxiliaryOperations {
		problems = append(problems, fmt.Sprintf("unsupported %s auxiliary operation %s", operation.Phase, operation.MethodName))
	}
	return problems
}

func unsupportedFollowUpHelpers(semantics *Semantics) []string {
	var problems []string
	for phase, followUp := range map[string]FollowUpSemantics{
		"create": semantics.CreateFollowUp,
		"update": semantics.UpdateFollowUp,
		"delete": semantics.DeleteFollowUp,
	} {
		for _, hook := range followUp.Hooks {
			if !supportedFormalHelper(hook.Helper) {
				problems = append(problems, fmt.Sprintf("%s helper %q is not supported", phase, hook.Helper))
			}
		}
	}
	return problems
}

func invalidListSemanticsProblems(semantics *Semantics) []string {
	if semantics.List == nil || strings.TrimSpace(semantics.List.ResponseItemsField) != "" {
		return nil
	}
	return []string{"list semantics require responseItemsField"}
}

func invalidDeleteSemanticsProblems(semantics *Semantics) []string {
	if semantics.Delete.Policy != "required" || len(semantics.Delete.TerminalStates) > 0 {
		return nil
	}
	return []string{"required delete semantics need terminal states"}
}

func supportedFormalHelper(helper string) bool {
	switch strings.TrimSpace(helper) {
	case "", "tfresource.CreateResource", "tfresource.UpdateResource", "tfresource.DeleteResource", "tfresource.WaitForUpdatedState", "tfresource.WaitForWorkRequestWithErrorHandling":
		return true
	default:
		return false
	}
}

func normalizeOCIError(err error) error {
	var serviceErr common.ServiceError
	if !errors.As(err, &serviceErr) {
		return err
	}
	if _, normalized := errorutil.OciErrorTypeResponse(err); normalized != nil {
		return normalized
	}
	return err
}

func isNotFound(err error) bool {
	if errors.Is(err, errResourceNotFound) {
		return true
	}
	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		if serviceErr.GetHTTPStatusCode() == 404 {
			return true
		}
		switch serviceErr.GetCode() {
		case "NotFound", "NotAuthorizedOrNotFound":
			return true
		default:
			return false
		}
	}

	message := err.Error()
	if strings.Contains(message, "http status code: 404") {
		return true
	}
	if strings.Contains(message, "NotFound") || strings.Contains(message, "NotAuthorizedOrNotFound") {
		return true
	}
	return false
}

func isConflict(err error) bool {
	if err == nil {
		return false
	}

	var serviceErr common.ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr.GetHTTPStatusCode() == 409
	}

	var conflictErr errorutil.ConflictOciError
	if errors.As(err, &conflictErr) {
		return true
	}

	return strings.Contains(err.Error(), "http status code: 409")
}

func (c ServiceClient[T]) selectListItem(body any, criteria map[string]any, preferredID string, phase readPhase) (any, error) {
	items, err := listItems(body, c.listResponseItemsField())
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errResourceNotFound
	}
	if c.config.Semantics != nil && c.config.Semantics.List != nil {
		return c.selectFormalListItem(items, criteriaWithPreferredID(criteria, preferredID), preferredID, phase)
	}
	return c.selectHeuristicListItem(items, criteriaWithPreferredID(criteria, preferredID))
}

func (c ServiceClient[T]) listResponseItemsField() string {
	if c.config.Semantics == nil || c.config.Semantics.List == nil {
		return ""
	}
	return c.config.Semantics.List.ResponseItemsField
}

func criteriaWithPreferredID(criteria map[string]any, preferredID string) map[string]any {
	if preferredID == "" {
		return criteria
	}
	cloned := make(map[string]any, len(criteria)+2)
	for key, value := range criteria {
		cloned[key] = value
	}
	cloned["id"] = preferredID
	cloned["ocid"] = preferredID
	return cloned
}

func (c ServiceClient[T]) selectHeuristicListItem(items []any, criteria map[string]any) (any, error) {
	targetID := firstNonEmpty(criteria, "ocid", "id")
	targetName := firstNonEmpty(criteria, "name", "metadataName")
	targetDisplayName := firstNonEmpty(criteria, "displayName")
	matches := heuristicListMatches(items, targetID, targetName, targetDisplayName)

	switch {
	case len(matches) == 1:
		return matches[0], nil
	case len(matches) > 1:
		return nil, fmt.Errorf("%s list response returned multiple matching resources", c.config.Kind)
	case len(items) == 1:
		return items[0], nil
	default:
		return nil, errResourceNotFound
	}
}

func heuristicListMatches(items []any, targetID string, targetName string, targetDisplayName string) []any {
	var matches []any
	for _, item := range items {
		values := jsonMap(item)
		switch {
		case targetID != "" && targetID == firstNonEmpty(values, "id", "ocid"):
			matches = append(matches, item)
		case targetDisplayName != "" && targetDisplayName == firstNonEmpty(values, "displayName"):
			matches = append(matches, item)
		case targetName != "" && targetName == firstNonEmpty(values, "name"):
			matches = append(matches, item)
		}
	}
	return matches
}

func (c ServiceClient[T]) selectFormalListItem(items []any, criteria map[string]any, preferredID string, phase readPhase) (any, error) {
	matches, comparedAny := c.formalListMatches(items, criteria, preferredID)
	matches = c.filterPhaseMatches(matches, phase)
	return c.resolveFormalListMatch(matches, comparedAny, preferredID)
}

func (c ServiceClient[T]) formalListMatches(items []any, criteria map[string]any, preferredID string) ([]any, bool) {
	matchFields := append([]string(nil), c.config.Semantics.List.MatchFields...)
	var matches []any
	comparedAny := false

	for _, item := range items {
		matched, compared := formalListItemMatch(item, criteria, preferredID, matchFields)
		comparedAny = comparedAny || compared
		if matched {
			matches = append(matches, item)
		}
	}
	return matches, comparedAny
}

func formalListItemMatch(item any, criteria map[string]any, preferredID string, matchFields []string) (bool, bool) {
	values := jsonMap(item)
	if preferredID != "" && preferredID == firstNonEmpty(values, "id", "ocid") {
		return true, false
	}

	comparedFields := 0
	for _, field := range matchFields {
		expected, ok := lookupMeaningfulValue(criteria, field)
		if !ok {
			continue
		}
		comparedFields++
		actual, ok := lookupMeaningfulValue(values, field)
		if !ok || !valuesEqual(expected, actual) {
			return false, comparedFields > 0
		}
	}
	return comparedFields > 0, comparedFields > 0
}

func (c ServiceClient[T]) resolveFormalListMatch(matches []any, comparedAny bool, preferredID string) (any, error) {
	switch {
	case len(matches) == 1:
		return matches[0], nil
	case len(matches) > 1:
		return nil, fmt.Errorf("%s formal list semantics returned multiple matching resources", c.config.Kind)
	case comparedAny || preferredID != "":
		return nil, errResourceNotFound
	default:
		return nil, fmt.Errorf("%s formal list semantics did not yield any match criteria", c.config.Kind)
	}
}

func (c ServiceClient[T]) filterPhaseMatches(matches []any, phase readPhase) []any {
	if len(matches) == 0 {
		return nil
	}

	switch phase {
	case readPhaseCreate, readPhaseUpdate, readPhaseObserve:
		filtered := make([]any, 0, len(matches))
		for _, item := range matches {
			if c.allowListItemForReadPhase(item, phase) {
				filtered = append(filtered, item)
			}
		}
		return filtered
	case readPhaseDelete:
		bestPriority := 0
		filtered := make([]any, 0, len(matches))
		for _, item := range matches {
			priority := c.deleteListItemPriority(item)
			if priority > bestPriority {
				bestPriority = priority
				filtered = filtered[:0]
			}
			if priority == bestPriority {
				filtered = append(filtered, item)
			}
		}
		return filtered
	default:
		return matches
	}
}

func (c ServiceClient[T]) allowListItemForReadPhase(item any, phase readPhase) bool {
	switch c.listItemLifecycleCategory(item) {
	case lifecycleCategoryProvisioning, lifecycleCategoryUpdating, lifecycleCategoryActive, lifecycleCategoryEmpty:
		return true
	case lifecycleCategoryUnknown:
		return phase == readPhaseObserve
	default:
		return false
	}
}

func (c ServiceClient[T]) deleteListItemPriority(item any) int {
	switch c.listItemLifecycleCategory(item) {
	case lifecycleCategoryProvisioning, lifecycleCategoryUpdating, lifecycleCategoryActive:
		return 4
	case lifecycleCategoryDeleting, lifecycleCategoryDeleted:
		return 3
	case lifecycleCategoryFailed:
		return 2
	default:
		return 1
	}
}

type lifecycleCategory string

const (
	lifecycleCategoryEmpty        lifecycleCategory = "empty"
	lifecycleCategoryProvisioning lifecycleCategory = "provisioning"
	lifecycleCategoryUpdating     lifecycleCategory = "updating"
	lifecycleCategoryActive       lifecycleCategory = "active"
	lifecycleCategoryDeleting     lifecycleCategory = "deleting"
	lifecycleCategoryDeleted      lifecycleCategory = "deleted"
	lifecycleCategoryFailed       lifecycleCategory = "failed"
	lifecycleCategoryUnknown      lifecycleCategory = "unknown"
)

func (c ServiceClient[T]) listItemLifecycleCategory(item any) lifecycleCategory {
	state := strings.ToUpper(firstNonEmpty(jsonMap(item), "lifecycleState", "status", "state"))
	if state == "" {
		return lifecycleCategoryEmpty
	}

	if category, ok := c.formalLifecycleCategory(state); ok {
		return category
	}
	return heuristicLifecycleCategory(state)
}

func (c ServiceClient[T]) formalLifecycleCategory(state string) (lifecycleCategory, bool) {
	if c.config.Semantics == nil {
		return "", false
	}

	switch {
	case containsString(c.config.Semantics.Lifecycle.ProvisioningStates, state):
		return lifecycleCategoryProvisioning, true
	case containsString(c.config.Semantics.Lifecycle.UpdatingStates, state):
		return lifecycleCategoryUpdating, true
	case containsString(c.config.Semantics.Lifecycle.ActiveStates, state):
		return lifecycleCategoryActive, true
	case containsString(c.config.Semantics.Delete.PendingStates, state):
		return lifecycleCategoryDeleting, true
	case containsString(c.config.Semantics.Delete.TerminalStates, state):
		return lifecycleCategoryDeleted, true
	default:
		return "", false
	}
}

func heuristicLifecycleCategory(state string) lifecycleCategory {
	switch {
	case strings.Contains(state, "FAIL"),
		strings.Contains(state, "ERROR"),
		strings.Contains(state, "NEEDS_ATTENTION"),
		strings.Contains(state, "INOPERABLE"):
		return lifecycleCategoryFailed
	case strings.Contains(state, "DELETED"),
		strings.Contains(state, "TERMINATED"):
		return lifecycleCategoryDeleted
	case strings.Contains(state, "DELETE"),
		strings.Contains(state, "TERMINAT"):
		return lifecycleCategoryDeleting
	case strings.Contains(state, "UPDAT"),
		strings.Contains(state, "MODIFY"),
		strings.Contains(state, "PATCH"):
		return lifecycleCategoryUpdating
	case strings.Contains(state, "CREATE"),
		strings.Contains(state, "PROVISION"),
		strings.Contains(state, "PENDING"),
		strings.Contains(state, "IN_PROGRESS"),
		strings.Contains(state, "ACCEPT"),
		strings.Contains(state, "START"):
		return lifecycleCategoryProvisioning
	default:
		return lifecycleCategoryUnknown
	}
}

func listItems(body any, responseItemsField string) ([]any, error) {
	value, err := listBodyStruct(body)
	if err != nil {
		return nil, err
	}
	if value.Kind() == reflect.Slice {
		return sliceValues(value), nil
	}

	if items, ok, err := configuredListItems(value, responseItemsField); ok || err != nil {
		return items, err
	}
	if items, ok := structSliceField(value, "Items"); ok {
		return items, nil
	}
	if items, ok := firstSliceListItems(value); ok {
		return items, nil
	}
	return nil, fmt.Errorf("OCI list body does not expose an items slice")
}

func sliceValues(value reflect.Value) []any {
	items := make([]any, 0, value.Len())
	for i := 0; i < value.Len(); i++ {
		items = append(items, value.Index(i).Interface())
	}
	return items
}

func valuesWithoutAliases(values map[string]any, aliases []string) map[string]any {
	filtered := make(map[string]any, len(values))
	for key, value := range values {
		if matchesAnyAlias(key, aliases) {
			continue
		}
		filtered[key] = value
	}
	return filtered
}

func matchesAnyAlias(key string, aliases []string) bool {
	for _, alias := range aliases {
		if strings.EqualFold(key, alias) || lowerCamel(key) == lowerCamel(alias) {
			return true
		}
	}
	return false
}

func assignField(field reflect.Value, raw any) error {
	converted, err := convertValue(raw, field.Type())
	if err != nil {
		return err
	}
	field.Set(converted)
	return nil
}

func convertValue(raw any, targetType reflect.Type) (reflect.Value, error) {
	if raw == nil {
		return reflect.Zero(targetType), nil
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("marshal source value: %w", err)
	}
	if targetType.Kind() == reflect.Interface {
		if converted, ok, err := convertPolymorphicInterfaceValue(payload, targetType); ok {
			return converted, err
		}
	}
	converted := reflect.New(targetType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return reflect.Value{}, fmt.Errorf("unmarshal into %s: %w", targetType, err)
	}
	return converted.Elem(), nil
}

func convertPolymorphicInterfaceValue(payload []byte, targetType reflect.Type) (reflect.Value, bool, error) {
	switch targetType {
	case autonomousDatabaseBaseType:
		body, err := convertAutonomousDatabaseBase(payload)
		if err != nil {
			return reflect.Value{}, true, err
		}
		converted := reflect.New(targetType).Elem()
		converted.Set(reflect.ValueOf(body))
		return converted, true, nil
	default:
		return reflect.Value{}, false, nil
	}
}

// OCI models CreateAutonomousDatabase with a polymorphic interface body. Resolve the CR spec into
// the matching concrete SDK type so request serialization uses the provider model instead of map[string]any.
//
//nolint:gocognit,gocyclo // The source discriminator maps to several concrete SDK request bodies in one switch.
func convertAutonomousDatabaseBase(payload []byte) (databasesdk.CreateAutonomousDatabaseBase, error) {
	source, err := jsonFieldString(payload, "source")
	if err != nil {
		return nil, fmt.Errorf("decode autonomous database source: %w", err)
	}

	concreteType, err := autonomousDatabaseBaseConcreteType(source)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(databasesdk.CreateAutonomousDatabaseBase)
	if !ok {
		return nil, fmt.Errorf("resolved CreateAutonomousDatabaseBase type %s does not implement the polymorphic interface", concreteType)
	}
	return body, nil
}

func autonomousDatabaseBaseConcreteType(source string) (reflect.Type, error) {
	switch strings.ToUpper(strings.TrimSpace(source)) {
	case "", "NONE":
		return reflect.TypeOf(databasesdk.CreateAutonomousDatabaseDetails{}), nil
	case "DATABASE":
		return reflect.TypeOf(databasesdk.CreateAutonomousDatabaseCloneDetails{}), nil
	case "CLONE_TO_REFRESHABLE":
		return reflect.TypeOf(databasesdk.CreateRefreshableAutonomousDatabaseCloneDetails{}), nil
	case "BACKUP_FROM_ID":
		return reflect.TypeOf(databasesdk.CreateAutonomousDatabaseFromBackupDetails{}), nil
	case "BACKUP_FROM_TIMESTAMP":
		return reflect.TypeOf(databasesdk.CreateAutonomousDatabaseFromBackupTimestampDetails{}), nil
	case "CROSS_REGION_DISASTER_RECOVERY":
		return reflect.TypeOf(databasesdk.CreateCrossRegionDisasterRecoveryDetails{}), nil
	case "CROSS_REGION_DATAGUARD":
		return reflect.TypeOf(databasesdk.CreateCrossRegionAutonomousDatabaseDataGuardDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported CreateAutonomousDatabaseBase source %q", source)
	}
}

func jsonFieldString(payload []byte, field string) (string, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(payload, &values); err != nil {
		return "", err
	}
	raw, ok := values[field]
	if !ok || string(raw) == "null" {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	return value, nil
}

func osokStatus(resource any) (*shared.OSOKStatus, error) {
	statusValue, err := statusStruct(resource)
	if err != nil {
		return nil, err
	}

	field := statusValue.FieldByName("OsokStatus")
	if !field.IsValid() || !field.CanAddr() {
		return nil, fmt.Errorf("resource %T does not expose Status.OsokStatus", resource)
	}

	status, ok := field.Addr().Interface().(*shared.OSOKStatus)
	if !ok {
		return nil, fmt.Errorf("resource %T Status.OsokStatus has unexpected type %T", resource, field.Addr().Interface())
	}
	return status, nil
}

func statusStruct(resource any) (reflect.Value, error) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return reflect.Value{}, err
	}
	field, ok := fieldValue(resourceValue, "Status")
	if !ok {
		return reflect.Value{}, fmt.Errorf("resource %T does not expose Status", resource)
	}
	return field, nil
}

func resourceStruct(resource any) (reflect.Value, error) {
	value := reflect.ValueOf(resource)
	if !value.IsValid() {
		return reflect.Value{}, fmt.Errorf("resource is nil")
	}
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return reflect.Value{}, fmt.Errorf("expected pointer resource, got %T", resource)
	}
	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("expected pointer to struct resource, got %T", resource)
	}
	return value, nil
}

func fieldValue(value reflect.Value, name string) (reflect.Value, bool) {
	field := value.FieldByName(name)
	if !field.IsValid() {
		return reflect.Value{}, false
	}
	return field, true
}

func fieldInterface(value reflect.Value, name string) any {
	field, ok := fieldValue(value, name)
	if !ok || !field.IsValid() {
		return nil
	}
	return field.Interface()
}

func lookupMetadataString(value reflect.Value, fieldName string) string {
	field, ok := fieldValue(value, fieldName)
	if !ok || field.Kind() != reflect.String {
		return ""
	}
	return field.String()
}

func assignDeterministicRetryToken(requestStruct reflect.Value, resource any) {
	field, ok := fieldValue(requestStruct, "OpcRetryToken")
	if !ok || !field.IsValid() || !field.CanSet() {
		return
	}

	switch field.Kind() {
	case reflect.Pointer:
		if !field.IsNil() {
			return
		}
	case reflect.String:
		if strings.TrimSpace(field.String()) != "" {
			return
		}
	default:
		return
	}

	token := resourceRetryToken(resource)
	if token == "" {
		return
	}
	_ = assignField(field, token)
}

func resourceRetryToken(resource any) string {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return ""
	}
	if uid := strings.TrimSpace(lookupMetadataString(resourceValue, "UID")); uid != "" {
		return uid
	}

	namespace := strings.TrimSpace(lookupMetadataString(resourceValue, "Namespace"))
	name := strings.TrimSpace(lookupMetadataString(resourceValue, "Name"))
	if namespace == "" && name == "" {
		return ""
	}

	sum := sha256.Sum256([]byte(namespace + "/" + name))
	return fmt.Sprintf("%x", sum[:16])
}

func resourceNamespace(resource any, fallback string) string {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return strings.TrimSpace(fallback)
	}
	namespace := lookupMetadataString(resourceValue, "Namespace")
	if strings.TrimSpace(namespace) != "" {
		return namespace
	}
	return strings.TrimSpace(fallback)
}

func fieldJSONName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

func jsonMap(value any) map[string]any {
	if value == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil
	}
	return decoded
}

func responseLifecycleState(response any) string {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return ""
	}
	return firstNonEmpty(jsonMap(body), "lifecycleState", "status")
}

func lookupValueByPaths(values map[string]any, paths ...string) (any, bool) {
	for _, path := range paths {
		if value, ok := lookupValueByPath(values, path); ok {
			return value, true
		}
	}
	return nil, false
}

func lookupMeaningfulValue(values map[string]any, path string) (any, bool) {
	value, ok := lookupValueByPath(values, path)
	if !ok || !meaningfulValue(value) {
		return nil, false
	}
	return value, true
}

func lookupValueByPath(values map[string]any, path string) (any, bool) {
	if values == nil {
		return nil, false
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, false
	}

	current := any(values)
	for _, segment := range strings.Split(path, ".") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return nil, false
		}

		mapValue, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := lookupMapKey(mapValue, segment)
		if !ok {
			return nil, false
		}
		current = next
	}

	return current, true
}

func lookupMapKey(values map[string]any, segment string) (any, bool) {
	if value, ok := values[segment]; ok {
		return value, true
	}

	normalized := normalizePathSegment(segment)
	for key, value := range values {
		if normalizePathSegment(key) == normalized {
			return value, true
		}
	}
	return nil, false
}

func meaningfulValue(value any) bool {
	if value == nil {
		return false
	}

	switch concrete := value.(type) {
	case string:
		return strings.TrimSpace(concrete) != ""
	case []any:
		return len(concrete) > 0
	case map[string]any:
		return len(concrete) > 0
	case bool:
		return concrete
	case float64:
		return concrete != 0
	default:
		return true
	}
}

func valuesEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return reflect.DeepEqual(left, right)
	}
	return string(leftPayload) == string(rightPayload)
}

func firstNonEmpty(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := lookupString(values, key); value != "" {
			return value
		}
	}
	return ""
}

func lookupString(values map[string]any, key string) string {
	raw, ok := lookupValueByPath(values, key)
	if !ok || raw == nil {
		return ""
	}
	switch concrete := raw.(type) {
	case string:
		return concrete
	default:
		return fmt.Sprint(concrete)
	}
}

func containsString(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func appendUniqueStrings(existing []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(extras))
	for _, value := range existing {
		seen[value] = struct{}{}
	}
	for _, value := range extras {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		existing = append(existing, value)
	}
	return existing
}

func lowerCamel(name string) string {
	tokens := splitCamel(name)
	if len(tokens) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(tokens[0])
	for _, token := range tokens[1:] {
		builder.WriteString(strings.ToUpper(token[:1]))
		builder.WriteString(token[1:])
	}
	return builder.String()
}

func splitCamel(name string) []string {
	if strings.TrimSpace(name) == "" {
		return nil
	}

	var tokens []string
	var current []rune
	runes := []rune(name)
	for index, r := range runes {
		if splitBeforeCamelRune(runes, index) {
			tokens = append(tokens, strings.ToLower(string(current)))
			current = current[:0]
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		tokens = append(tokens, strings.ToLower(string(current)))
	}
	return tokens
}

func listBodyStruct(body any) (reflect.Value, error) {
	value, ok := indirectValue(reflect.ValueOf(body))
	if !ok {
		return reflect.Value{}, errResourceNotFound
	}
	if value.Kind() != reflect.Struct && value.Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("OCI list body must be a struct or slice, got %T", body)
	}
	return value, nil
}

func configuredListItems(value reflect.Value, fieldName string) ([]any, bool, error) {
	fieldName = strings.TrimSpace(fieldName)
	if fieldName == "" {
		return nil, false, nil
	}

	itemsField := value.FieldByName(fieldName)
	if !itemsField.IsValid() {
		return nil, true, fmt.Errorf("OCI list body does not expose %s", fieldName)
	}
	if itemsField.Kind() != reflect.Slice {
		return nil, true, fmt.Errorf("OCI list body %s is not a slice", fieldName)
	}
	return sliceValues(itemsField), true, nil
}

func structSliceField(value reflect.Value, fieldName string) ([]any, bool) {
	itemsField := value.FieldByName(fieldName)
	if !itemsField.IsValid() || itemsField.Kind() != reflect.Slice {
		return nil, false
	}
	return sliceValues(itemsField), true
}

func firstSliceListItems(value reflect.Value) ([]any, bool) {
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if field.Kind() == reflect.Slice {
			return sliceValues(field), true
		}
	}
	return nil, false
}

func splitBeforeCamelRune(runes []rune, index int) bool {
	if index == 0 {
		return false
	}

	current := runes[index]
	prev := runes[index-1]
	nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
	return unicode.IsUpper(current) &&
		(unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextIsLower))
}
