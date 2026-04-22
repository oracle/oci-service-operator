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
	databasetoolssdk "github.com/oracle/oci-go-sdk/v65/databasetools"
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

const (
	defaultRequeueDuration = time.Minute

	asyncStrategyLifecycle   = "lifecycle"
	asyncStrategyNone        = "none"
	asyncStrategyWorkRequest = "workrequest"

	asyncRuntimeGeneratedRuntime = "generatedruntime"
	asyncRuntimeHandwritten      = "handwritten"

	asyncWorkRequestSourceServiceSDK      = "service-sdk"
	asyncWorkRequestSourceWorkRequestsAPI = "workrequests-service"
	asyncWorkRequestSourceProviderHelper  = "provider-helper"

	asyncPhaseCreate = "create"
	asyncPhaseUpdate = "update"
	asyncPhaseDelete = "delete"
)

var errResourceNotFound = errors.New("generated runtime resource not found")

var (
	passwordSourceType                       = reflect.TypeOf(shared.PasswordSource{})
	usernameSourceType                       = reflect.TypeOf(shared.UsernameSource{})
	autonomousDatabaseBaseType               = reflect.TypeOf((*databasesdk.CreateAutonomousDatabaseBase)(nil)).Elem()
	databaseToolsConnectionCreateDetailsType = reflect.TypeOf((*databasetoolssdk.CreateDatabaseToolsConnectionDetails)(nil)).Elem()
	databaseToolsConnectionUpdateDetailsType = reflect.TypeOf((*databasetoolssdk.UpdateDatabaseToolsConnectionDetails)(nil)).Elem()
)

type createContextKey string

const (
	skipExistingBeforeCreateContextKey createContextKey = "generatedruntime/skip-existing-before-create"
	lookupSpecRootKey                                   = "__generatedruntime_lookup_spec_root__"
	lookupStatusRootKey                                 = "__generatedruntime_lookup_status_root__"
)

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

type ExistingBeforeCreateDecision string

const (
	ExistingBeforeCreateDecisionAllow ExistingBeforeCreateDecision = "allow"
	ExistingBeforeCreateDecisionSkip  ExistingBeforeCreateDecision = "skip"
	ExistingBeforeCreateDecisionFail  ExistingBeforeCreateDecision = "fail"
)

type Hook struct {
	Helper     string
	EntityType string
	Action     string
}

type AsyncSemantics struct {
	Strategy             string
	Runtime              string
	FormalClassification string
	WorkRequest          *WorkRequestSemantics
}

type WorkRequestSemantics struct {
	Source            string
	Phases            []string
	LegacyFieldBridge *WorkRequestLegacyFieldBridge
}

type WorkRequestLegacyFieldBridge struct {
	Create string
	Update string
	Delete string
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

type IdentityHooks[T any] struct {
	Resolve                   func(T) (any, error)
	RecordPath                func(T, any)
	RecordTracked             func(T, any, string)
	GuardExistingBeforeCreate func(context.Context, T) (ExistingBeforeCreateDecision, error)
	LookupExisting            func(context.Context, T, any) (any, error)
	SeedSyntheticTrackedID    func(T, any) func()
}

type ReadHooks struct {
	Get  *Operation
	List *Operation
}

type TrackedRecreateHooks[T any] struct {
	ClearTrackedIdentity func(T)
}

type StatusHooks[T any] struct {
	ClearProjectedStatus   func(T) any
	ShouldRestoreOnFailure func(T, any) bool
	RestoreStatus          func(T, any)
	ProjectStatus          func(T, any) error
	ApplyLifecycle         func(T, any) (servicemanager.OSOKResponse, error)
	MarkDeleted            func(T, string)
	MarkTerminating        func(T, any)
}

type ParityHooks[T any] struct {
	NormalizeDesiredState   func(T, any)
	ValidateCreateOnlyDrift func(T, any) error
	RequiresParityHandling  func(T, any) bool
	ApplyParityUpdate       func(context.Context, T, any) (servicemanager.OSOKResponse, error)
}

// AsyncHooks is the bounded work-request seam for generatedruntime-owned async
// flows. Implementations should normalize service-local work-request state via
// servicemanager.BuildWorkRequestAsyncOperation and recover the target resource
// identity without widening generatedruntime into a generic provider contract.
type AsyncHooks[T any] struct {
	Adapter           servicemanager.WorkRequestAsyncAdapter
	GetWorkRequest    func(context.Context, string) (any, error)
	ResolveAction     func(any) (string, error)
	ResolvePhase      func(any) (shared.OSOKAsyncPhase, bool, error)
	RecoverResourceID func(T, any, shared.OSOKAsyncPhase) (string, error)
	Message           func(shared.OSOKAsyncPhase, any) string
}

type DeleteConfirmStage string

const (
	DeleteConfirmStageAlreadyPending DeleteConfirmStage = "already-pending"
	DeleteConfirmStageAfterRequest   DeleteConfirmStage = "after-request"
)

type DeleteOutcome struct {
	Handled bool
	Deleted bool
}

// DeleteHooks is the bounded delete-only seam for generatedruntime confirm-delete
// flows. Implementations may override delete rereads, normalize or project
// delete-phase errors, and short-circuit delete outcome handling without
// widening create, update, or observe into generic provider hooks.
type DeleteHooks[T any] struct {
	ConfirmRead  func(context.Context, T, string) (any, error)
	HandleError  func(T, error) error
	ApplyOutcome func(T, any, DeleteConfirmStage) (DeleteOutcome, error)
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
	UpdateCandidate []string
	Mutable         []string
	ForceNew        []string
	ConflictsWith   map[string][]string
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
	Async               *AsyncSemantics
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
	Identity         IdentityHooks[T]
	Read             ReadHooks
	TrackedRecreate  TrackedRecreateHooks[T]
	StatusHooks      StatusHooks[T]
	ParityHooks      ParityHooks[T]
	Async            AsyncHooks[T]
	DeleteHooks      DeleteHooks[T]
	BuildCreateBody  func(context.Context, T, string) (any, error)
	BuildUpdateBody  func(context.Context, T, string, any) (any, bool, error)

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
	identity                  any
	currentID                 string
	liveResponse              any
	restoreSyntheticTrackedID func()
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
	if err := validateGeneratedWorkRequestAsyncHooks(cfg); err != nil {
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

func (c ServiceClient[T]) workRequestLegacyBridge() servicemanager.WorkRequestLegacyBridge {
	workRequest := c.generatedWorkRequestSemantics()
	if workRequest == nil || workRequest.LegacyFieldBridge == nil {
		return servicemanager.WorkRequestLegacyBridge{}
	}
	return servicemanager.WorkRequestLegacyBridge{
		Create: workRequest.LegacyFieldBridge.Create,
		Update: workRequest.LegacyFieldBridge.Update,
		Delete: workRequest.LegacyFieldBridge.Delete,
	}
}

func (c ServiceClient[T]) generatedWorkRequestSemantics() *WorkRequestSemantics {
	if c.config.Semantics == nil || c.config.Semantics.Async == nil {
		return nil
	}
	async := c.config.Semantics.Async
	if strings.TrimSpace(async.Strategy) != asyncStrategyWorkRequest ||
		strings.TrimSpace(async.Runtime) != asyncRuntimeGeneratedRuntime {
		return nil
	}
	return async.WorkRequest
}

func (c ServiceClient[T]) generatedWorkRequestAsyncEnabled() bool {
	return c.generatedWorkRequestSemantics() != nil
}

func (c ServiceClient[T]) generatedWorkRequestPhaseEnabled(phase shared.OSOKAsyncPhase) bool {
	workRequest := c.generatedWorkRequestSemantics()
	if workRequest == nil || phase == "" {
		return false
	}
	for _, rawPhase := range workRequest.Phases {
		if strings.TrimSpace(rawPhase) == string(phase) {
			return true
		}
	}
	return false
}

func (c ServiceClient[T]) defaultConfiguredWorkRequestPhase() shared.OSOKAsyncPhase {
	workRequest := c.generatedWorkRequestSemantics()
	if workRequest == nil || len(workRequest.Phases) != 1 {
		return ""
	}
	return shared.OSOKAsyncPhase(strings.TrimSpace(workRequest.Phases[0]))
}

func (c ServiceClient[T]) currentGeneratedWorkRequest(resource T) (string, shared.OSOKAsyncPhase) {
	if !c.generatedWorkRequestAsyncEnabled() {
		return "", ""
	}

	status, err := osokStatus(resource)
	if err != nil {
		return "", ""
	}
	workRequestID, phase := servicemanager.ResolveTrackedWorkRequest(
		status,
		resource,
		c.workRequestLegacyBridge(),
		c.defaultConfiguredWorkRequestPhase(),
	)
	if workRequestID == "" || phase == "" {
		return "", ""
	}
	return workRequestID, phase
}

func (c ServiceClient[T]) applyAsyncOperation(status *shared.OSOKStatus, resource T, current *shared.OSOKAsyncOperation) servicemanager.AsyncProjection {
	if c.generatedWorkRequestAsyncEnabled() {
		return servicemanager.ApplyAsyncOperationWithLegacyBridge(status, resource, c.workRequestLegacyBridge(), current, c.config.Log)
	}
	return servicemanager.ApplyAsyncOperation(status, current, c.config.Log)
}

func (c ServiceClient[T]) clearAsyncOperation(status *shared.OSOKStatus, resource T) {
	if c.generatedWorkRequestAsyncEnabled() {
		servicemanager.ClearAsyncOperationWithLegacyBridge(status, resource, c.workRequestLegacyBridge())
		return
	}
	servicemanager.ClearAsyncOperation(status)
}

func (c ServiceClient[T]) getReadOperation() *Operation {
	if c.config.Read.Get != nil {
		return c.config.Read.Get
	}
	return c.config.Get
}

func (c ServiceClient[T]) listReadOperation() *Operation {
	if c.config.Read.List != nil {
		return c.config.Read.List
	}
	return c.config.List
}

func (c ServiceClient[T]) hasReadableOperation() bool {
	return c.getReadOperation() != nil || c.listReadOperation() != nil
}

func (c ServiceClient[T]) hasDeleteConfirmRead() bool {
	return c.config.DeleteHooks.ConfirmRead != nil || c.hasReadableOperation()
}

func (c ServiceClient[T]) prepareIdentity(resource T) (any, error) {
	if c.config.Identity.Resolve == nil {
		return nil, nil
	}

	identity, err := c.config.Identity.Resolve(resource)
	if err != nil {
		return nil, err
	}
	if c.config.Identity.RecordPath != nil {
		c.config.Identity.RecordPath(resource, identity)
	}
	return identity, nil
}

func (c ServiceClient[T]) lookupExistingByIdentity(ctx context.Context, resource T, identity any) (any, error) {
	if c.config.Identity.LookupExisting == nil || identity == nil {
		return nil, nil
	}

	response, err := c.config.Identity.LookupExisting(ctx, resource, identity)
	switch {
	case err == nil:
		return response, nil
	case errors.Is(err, errResourceNotFound), isReadNotFound(err):
		return nil, nil
	default:
		return nil, err
	}
}

func (c ServiceClient[T]) guardExistingBeforeCreate(ctx context.Context, resource T) (ExistingBeforeCreateDecision, error) {
	if c.config.Identity.GuardExistingBeforeCreate == nil {
		return ExistingBeforeCreateDecisionAllow, nil
	}

	decision, err := c.config.Identity.GuardExistingBeforeCreate(ctx, resource)
	if err != nil {
		return ExistingBeforeCreateDecisionFail, err
	}
	if decision == "" {
		decision = ExistingBeforeCreateDecisionAllow
	}

	switch decision {
	case ExistingBeforeCreateDecisionAllow, ExistingBeforeCreateDecisionSkip:
		return decision, nil
	case ExistingBeforeCreateDecisionFail:
		return decision, fmt.Errorf("%s identity guard rejected pre-create reuse", c.config.Kind)
	default:
		return "", fmt.Errorf("%s identity guard returned unsupported pre-create decision %q", c.config.Kind, decision)
	}
}

func (c ServiceClient[T]) recordTrackedIdentity(resource T, identity any, resourceID string) {
	if c.config.Identity.RecordTracked == nil {
		return
	}
	c.config.Identity.RecordTracked(resource, identity, resourceID)
}

func (c ServiceClient[T]) clearTrackedIdentity(resource T) {
	if c.config.TrackedRecreate.ClearTrackedIdentity == nil {
		return
	}
	c.config.TrackedRecreate.ClearTrackedIdentity(resource)
}

func (c ServiceClient[T]) normalizeDesiredState(resource T, currentResponse any) {
	if c.config.ParityHooks.NormalizeDesiredState == nil {
		return
	}
	c.config.ParityHooks.NormalizeDesiredState(resource, currentResponse)
}

func (c ServiceClient[T]) clearProjectedStatus(resource T) (any, bool) {
	if c.config.StatusHooks.ClearProjectedStatus == nil {
		return nil, false
	}
	return c.config.StatusHooks.ClearProjectedStatus(resource), true
}

func (c ServiceClient[T]) confirmDeleteRead(ctx context.Context, resource T, currentID string) (any, error) {
	if c.config.DeleteHooks.ConfirmRead != nil {
		response, err := c.config.DeleteHooks.ConfirmRead(ctx, resource, currentID)
		if err != nil {
			return nil, c.handleDeleteError(resource, err)
		}
		return response, nil
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		return nil, c.handleDeleteError(resource, err)
	}
	return response, nil
}

func (c ServiceClient[T]) handleDeleteError(resource T, err error) error {
	if err == nil {
		return nil
	}
	if c.config.DeleteHooks.HandleError != nil {
		if handledErr := c.config.DeleteHooks.HandleError(resource, err); handledErr != nil {
			return handledErr
		}
	}
	return err
}

func (c ServiceClient[T]) applyDeleteOutcomeHooks(resource T, response any, stage DeleteConfirmStage) (DeleteOutcome, error) {
	if c.config.DeleteHooks.ApplyOutcome == nil {
		return DeleteOutcome{}, nil
	}
	return c.config.DeleteHooks.ApplyOutcome(resource, response, stage)
}

func (c ServiceClient[T]) restoreStatusAfterFailure(resource T, baseline any) {
	if c.config.StatusHooks.RestoreStatus == nil {
		return
	}
	if c.config.StatusHooks.ShouldRestoreOnFailure != nil && !c.config.StatusHooks.ShouldRestoreOnFailure(resource, baseline) {
		return
	}
	c.config.StatusHooks.RestoreStatus(resource, baseline)
}

func (c ServiceClient[T]) startGeneratedWorkRequest(resource T, response any, phase shared.OSOKAsyncPhase, identity any) (string, error) {
	if err := c.projectStatusWithHooks(resource, response); err != nil {
		return "", err
	}
	if resourceID := responseID(response); resourceID != "" {
		if c.config.Identity.RecordTracked != nil {
			c.recordTrackedIdentity(resource, identity, resourceID)
		} else if status, err := osokStatus(resource); err == nil {
			status.Ocid = shared.OCID(resourceID)
		}
	}
	c.seedOpeningWorkRequestID(resource, response, phase)
	workRequestID, resolvedPhase := c.currentGeneratedWorkRequest(resource)
	if workRequestID != "" && resolvedPhase == phase {
		return workRequestID, nil
	}
	if workRequestID == "" {
		return "", fmt.Errorf("%s %s did not return an opc-work-request-id", c.config.Kind, phase)
	}
	return "", fmt.Errorf("%s %s returned work request %s for unexpected phase %q", c.config.Kind, phase, workRequestID, resolvedPhase)
}

func (c ServiceClient[T]) fetchGeneratedWorkRequest(ctx context.Context, workRequestID string) (any, error) {
	if c.config.Async.GetWorkRequest == nil {
		return nil, fmt.Errorf("%s workrequest async hooks require GetWorkRequest", c.config.Kind)
	}
	workRequest, err := c.config.Async.GetWorkRequest(ctx, strings.TrimSpace(workRequestID))
	if err != nil {
		return nil, normalizeOCIError(err)
	}
	if workRequest == nil {
		return nil, fmt.Errorf("%s work request %s did not return a body payload", c.config.Kind, workRequestID)
	}
	return workRequest, nil
}

func (c ServiceClient[T]) buildGeneratedWorkRequestAsyncOperation(resource T, workRequest any, explicitPhase shared.OSOKAsyncPhase) (*shared.OSOKAsyncOperation, error) {
	status, err := osokStatus(resource)
	if err != nil {
		return nil, err
	}

	rawAction := ""
	if c.config.Async.ResolveAction != nil {
		rawAction, err = c.config.Async.ResolveAction(workRequest)
		if err != nil {
			return nil, err
		}
	}

	fallbackPhase := explicitPhase
	if fallbackPhase == "" {
		_, fallbackPhase = c.currentGeneratedWorkRequest(resource)
	}
	if fallbackPhase == "" {
		fallbackPhase = c.defaultConfiguredWorkRequestPhase()
	}
	if c.config.Async.ResolvePhase != nil {
		derivedPhase, ok, err := c.config.Async.ResolvePhase(workRequest)
		if err != nil {
			return nil, err
		}
		if ok {
			if fallbackPhase != "" && fallbackPhase != derivedPhase {
				return nil, fmt.Errorf(
					"%s work request %s exposes phase %q while reconcile expected %q",
					c.config.Kind,
					workRequestStringField(workRequest, "Id"),
					derivedPhase,
					fallbackPhase,
				)
			}
			fallbackPhase = derivedPhase
		}
	}

	current, err := servicemanager.BuildWorkRequestAsyncOperation(status, c.config.Async.Adapter, servicemanager.WorkRequestAsyncInput{
		RawStatus:        workRequestStringField(workRequest, "Status"),
		RawAction:        rawAction,
		RawOperationType: workRequestStringField(workRequest, "OperationType"),
		WorkRequestID:    workRequestStringField(workRequest, "Id"),
		PercentComplete:  workRequestFloat32Field(workRequest, "PercentComplete"),
		FallbackPhase:    fallbackPhase,
	})
	if err != nil {
		return nil, err
	}

	message := strings.TrimSpace(c.generatedWorkRequestMessage(current.Phase, workRequest))
	if message != "" {
		current.Message = message
	}
	return current, nil
}

func (c ServiceClient[T]) generatedWorkRequestMessage(phase shared.OSOKAsyncPhase, workRequest any) string {
	if c.config.Async.Message != nil {
		if message := strings.TrimSpace(c.config.Async.Message(phase, workRequest)); message != "" {
			return message
		}
	}
	workRequestID := workRequestStringField(workRequest, "Id")
	rawStatus := workRequestStringField(workRequest, "Status")
	switch {
	case phase != "" && workRequestID != "" && rawStatus != "":
		return fmt.Sprintf("%s %s work request %s is %s", c.config.Kind, phase, workRequestID, rawStatus)
	case phase != "" && rawStatus != "":
		return fmt.Sprintf("%s %s work request is %s", c.config.Kind, phase, rawStatus)
	default:
		return ""
	}
}

func (c ServiceClient[T]) markWorkRequestOperation(resource T, current *shared.OSOKAsyncOperation) servicemanager.OSOKResponse {
	status, err := osokStatus(resource)
	if err != nil {
		return servicemanager.OSOKResponse{IsSuccessful: false}
	}

	now := metav1.Now()
	if currentID := c.currentID(resource); currentID != "" {
		status.Ocid = shared.OCID(currentID)
	}
	if status.Ocid != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	if current != nil && current.UpdatedAt == nil {
		current.UpdatedAt = &now
	}

	projection := c.applyAsyncOperation(status, resource, current)
	return servicemanager.OSOKResponse{
		IsSuccessful:    projection.Condition != shared.Failed,
		ShouldRequeue:   projection.ShouldRequeue,
		RequeueDuration: defaultRequeueDuration,
	}
}

func (c ServiceClient[T]) setWorkRequestOperation(resource T, current *shared.OSOKAsyncOperation, class shared.OSOKAsyncNormalizedClass, message string) servicemanager.OSOKResponse {
	next := *current
	next.NormalizedClass = class
	next.Message = strings.TrimSpace(message)
	next.UpdatedAt = nil
	return c.markWorkRequestOperation(resource, &next)
}

func (c ServiceClient[T]) failWorkRequestOperation(resource T, current *shared.OSOKAsyncOperation, err error) (servicemanager.OSOKResponse, error) {
	if current == nil {
		return c.failCreateOrUpdate(resource, err)
	}
	c.recordErrorRequestID(resource, err)

	class := current.NormalizedClass
	switch class {
	case shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
	default:
		class = shared.OSOKAsyncClassFailed
	}

	return c.setWorkRequestOperation(resource, current, class, err.Error()), err
}

func (c ServiceClient[T]) resumeGeneratedWorkRequestCreateOrUpdate(
	ctx context.Context,
	resource T,
	identity any,
	workRequestID string,
	phase shared.OSOKAsyncPhase,
) (servicemanager.OSOKResponse, error) {
	workRequest, err := c.fetchGeneratedWorkRequest(ctx, workRequestID)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}

	currentAsync, err := c.buildGeneratedWorkRequestAsyncOperation(resource, workRequest, phase)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		return c.setWorkRequestOperation(resource, currentAsync, shared.OSOKAsyncClassPending, c.generatedWorkRequestMessage(currentAsync.Phase, workRequest)), nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		return c.failWorkRequestOperation(
			resource,
			currentAsync,
			fmt.Errorf("%s %s work request %s finished with status %s", c.config.Kind, currentAsync.Phase, workRequestID, currentAsync.RawStatus),
		)
	case shared.OSOKAsyncClassSucceeded:
		return c.completeGeneratedWorkRequestWrite(ctx, resource, identity, workRequest, currentAsync)
	default:
		return c.failWorkRequestOperation(
			resource,
			currentAsync,
			fmt.Errorf("%s %s work request %s projected unsupported async class %s", c.config.Kind, currentAsync.Phase, workRequestID, currentAsync.NormalizedClass),
		)
	}
}

func (c ServiceClient[T]) completeGeneratedWorkRequestWrite(
	ctx context.Context,
	resource T,
	identity any,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) (servicemanager.OSOKResponse, error) {
	resourceID, err := c.resolveGeneratedWorkRequestResourceID(resource, workRequest, current.Phase)
	if err != nil {
		return c.failWorkRequestOperation(resource, current, err)
	}

	response, err := c.readResource(ctx, resource, resourceID, phaseReadPhase(string(current.Phase)))
	if err != nil {
		if current.Phase == shared.OSOKAsyncPhaseCreate && errors.Is(err, errResourceNotFound) {
			return c.setWorkRequestOperation(
				resource,
				current,
				shared.OSOKAsyncClassPending,
				fmt.Sprintf("%s create work request %s succeeded; waiting for %s %s to become readable", c.config.Kind, current.WorkRequestID, c.config.Kind, resourceID),
			), nil
		}
		return c.failWorkRequestOperation(resource, current, c.generatedWorkRequestReadError(current, resourceID, err))
	}

	return c.applySuccessWithIdentity(resource, response, fallbackConditionForAsyncPhase(current.Phase), identity)
}

func (c ServiceClient[T]) generatedWorkRequestReadError(current *shared.OSOKAsyncOperation, resourceID string, err error) error {
	if !errors.Is(err, errResourceNotFound) {
		return err
	}
	switch current.Phase {
	case shared.OSOKAsyncPhaseUpdate:
		return fmt.Errorf("%s update work request %s succeeded but %s %s is no longer readable", c.config.Kind, current.WorkRequestID, c.config.Kind, resourceID)
	case shared.OSOKAsyncPhaseDelete:
		return fmt.Errorf("%s delete work request %s succeeded but %s %s is still unresolved", c.config.Kind, current.WorkRequestID, c.config.Kind, resourceID)
	default:
		return err
	}
}

func (c ServiceClient[T]) resolveGeneratedWorkRequestResourceID(resource T, workRequest any, phase shared.OSOKAsyncPhase) (string, error) {
	if resourceID := strings.TrimSpace(c.currentID(resource)); resourceID != "" {
		return resourceID, nil
	}
	if c.config.Async.RecoverResourceID != nil {
		resourceID, err := c.config.Async.RecoverResourceID(resource, workRequest, phase)
		if err != nil {
			return "", err
		}
		if resourceID := strings.TrimSpace(resourceID); resourceID != "" {
			return resourceID, nil
		}
	}
	return "", fmt.Errorf("%s %s work request %s did not expose a %s identifier", c.config.Kind, phase, workRequestStringField(workRequest, "Id"), c.config.Kind)
}

func (c ServiceClient[T]) resumeGeneratedWorkRequestDelete(
	ctx context.Context,
	resource T,
	workRequestID string,
) (bool, error) {
	workRequest, err := c.fetchGeneratedWorkRequest(ctx, workRequestID)
	if err != nil {
		c.recordErrorRequestID(resource, err)
		return false, err
	}

	currentAsync, err := c.buildGeneratedWorkRequestAsyncOperation(resource, workRequest, shared.OSOKAsyncPhaseDelete)
	if err != nil {
		return false, err
	}

	switch currentAsync.NormalizedClass {
	case shared.OSOKAsyncClassPending:
		_ = c.setWorkRequestOperation(resource, currentAsync, shared.OSOKAsyncClassPending, c.generatedWorkRequestMessage(currentAsync.Phase, workRequest))
		return false, nil
	case shared.OSOKAsyncClassFailed, shared.OSOKAsyncClassCanceled, shared.OSOKAsyncClassAttention, shared.OSOKAsyncClassUnknown:
		_, err := c.failWorkRequestOperation(
			resource,
			currentAsync,
			fmt.Errorf("%s %s work request %s finished with status %s", c.config.Kind, currentAsync.Phase, workRequestID, currentAsync.RawStatus),
		)
		return false, err
	case shared.OSOKAsyncClassSucceeded:
		return c.completeGeneratedWorkRequestDelete(ctx, resource, workRequest, currentAsync)
	default:
		_, err := c.failWorkRequestOperation(
			resource,
			currentAsync,
			fmt.Errorf("%s %s work request %s projected unsupported async class %s", c.config.Kind, currentAsync.Phase, workRequestID, currentAsync.NormalizedClass),
		)
		return false, err
	}
}

func (c ServiceClient[T]) completeGeneratedWorkRequestDelete(
	ctx context.Context,
	resource T,
	workRequest any,
	current *shared.OSOKAsyncOperation,
) (bool, error) {
	currentID := strings.TrimSpace(c.currentID(resource))
	if currentID == "" && c.config.Async.RecoverResourceID != nil {
		recoveredID, err := c.config.Async.RecoverResourceID(resource, workRequest, shared.OSOKAsyncPhaseDelete)
		if err == nil {
			currentID = strings.TrimSpace(recoveredID)
		}
	}
	if currentID == "" {
		c.markDeletedWithHooks(resource, fmt.Sprintf("OCI %s delete work request completed", c.config.Kind))
		return true, nil
	}

	response, err := c.readResource(ctx, resource, currentID, readPhaseDelete)
	if err != nil {
		if isDeleteNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.recordErrorRequestID(resource, err)
			c.markDeletedWithHooks(resource, "OCI resource deleted")
			return true, nil
		}
		_, deleteErr := c.failWorkRequestOperation(resource, current, err)
		return false, deleteErr
	}

	if err := c.projectStatusWithHooks(resource, response); err != nil {
		return false, err
	}
	if semantics := c.config.Semantics; semantics != nil {
		return c.applyDeletePolicy(resource, response, semantics)
	}
	if err := c.markTerminatingWithHooks(resource, response); err != nil {
		return false, err
	}
	return false, nil
}

func (c ServiceClient[T]) CreateOrUpdate(ctx context.Context, resource T, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	if response, err, handled := c.validateCreateOrUpdateRequest(resource); handled {
		return response, err
	}

	identity, err := c.prepareIdentity(resource)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if workRequestID, phase := c.currentGeneratedWorkRequest(resource); workRequestID != "" {
		switch phase {
		case shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncPhaseUpdate:
			return c.resumeGeneratedWorkRequestCreateOrUpdate(ctx, resource, identity, workRequestID, phase)
		case shared.OSOKAsyncPhaseDelete:
			return c.failCreateOrUpdate(resource, fmt.Errorf("%s delete work request %s is still active during CreateOrUpdate", c.config.Kind, workRequestID))
		}
	}

	namespace := resourceNamespace(resource, req.Namespace)
	state, err := c.prepareCreateOrUpdateState(ctx, resource, identity)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if err := c.validateMutationPolicy(resource, state.currentID != "", state.liveResponse); err != nil {
		if state.restoreSyntheticTrackedID != nil {
			state.restoreSyntheticTrackedID()
		}
		return c.failCreateOrUpdate(resource, err)
	}

	var response servicemanager.OSOKResponse
	if response, err, handled := c.applyExistingResourceHooks(ctx, resource, state, namespace); handled {
		if err != nil {
			if state.restoreSyntheticTrackedID != nil {
				state.restoreSyntheticTrackedID()
			}
			return response, err
		}
		if state.restoreSyntheticTrackedID != nil && c.config.Identity.RecordTracked == nil {
			state.restoreSyntheticTrackedID()
		}
		return response, nil
	}

	statusBaseline, statusCleared := c.clearProjectedStatus(resource)
	if state.currentID != "" {
		response, err = c.reconcileExistingResource(ctx, resource, state, namespace)
	} else {
		response, err = c.createOrReadResource(ctx, resource, namespace, state.identity)
	}
	if err != nil {
		if statusCleared {
			c.restoreStatusAfterFailure(resource, statusBaseline)
		}
		if state.restoreSyntheticTrackedID != nil {
			state.restoreSyntheticTrackedID()
		}
		return response, err
	}
	if state.restoreSyntheticTrackedID != nil && c.config.Identity.RecordTracked == nil {
		state.restoreSyntheticTrackedID()
	}
	return response, nil
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

func (c ServiceClient[T]) prepareCreateOrUpdateState(ctx context.Context, resource T, identity any) (createOrUpdateState, error) {
	currentID, existingResponse, resolvedBeforeCreate, restoreSyntheticTrackedID, err := c.resolveCurrentResource(ctx, resource, identity)
	if err != nil {
		return createOrUpdateState{}, err
	}

	currentID, liveResponse, err := c.loadLiveMutationResponse(ctx, resource, currentID, existingResponse, resolvedBeforeCreate)
	if err != nil {
		if restoreSyntheticTrackedID != nil {
			restoreSyntheticTrackedID()
		}
		return createOrUpdateState{}, err
	}
	c.normalizeDesiredState(resource, liveResponse)

	return createOrUpdateState{
		identity:                  identity,
		currentID:                 currentID,
		liveResponse:              liveResponse,
		restoreSyntheticTrackedID: restoreSyntheticTrackedID,
	}, nil
}

func (c ServiceClient[T]) resolveCurrentResource(ctx context.Context, resource T, identity any) (string, any, bool, func(), error) {
	currentID := c.currentID(resource)
	existingResponse, trackedIDStale, err := c.resolveExistingBeforeCreate(ctx, resource, identity)
	if err != nil {
		return "", nil, false, nil, err
	}

	var restoreSyntheticTrackedID func()
	if existingResponse != nil && currentID == "" && c.config.Identity.SeedSyntheticTrackedID != nil {
		restoreSyntheticTrackedID = c.config.Identity.SeedSyntheticTrackedID(resource, identity)
		currentID = c.currentID(resource)
	}

	currentID, resolvedBeforeCreate := c.resolveTrackedCurrentID(resource, currentID, existingResponse, trackedIDStale)
	return currentID, existingResponse, resolvedBeforeCreate, restoreSyntheticTrackedID, nil
}

func (c ServiceClient[T]) resolveTrackedCurrentID(resource T, currentID string, existingResponse any, trackedIDStale bool) (string, bool) {
	originalCurrentID := currentID
	if trackedIDStale {
		c.clearTrackedIdentity(resource)
	}
	if existingResponse != nil {
		resolvedID := responseID(existingResponse)
		if currentID == "" && resolvedID != "" {
			return resolvedID, true
		}
		if trackedIDStale && resolvedID != "" && resolvedID != originalCurrentID {
			return resolvedID, true
		}
	}
	if trackedIDStale {
		return "", false
	}
	return currentID, false
}

func (c ServiceClient[T]) trackedStatusIDCanBeClearedAfterGetNotFound(resource T, preferredID string) bool {
	getOp := c.getReadOperation()
	if preferredID == "" || !c.usesStatusOnlyCurrentID(resource, preferredID) || getOp == nil {
		return false
	}

	values, err := lookupValues(resource)
	if err != nil {
		return false
	}

	if len(getOp.Fields) > 0 {
		for _, field := range getOp.Fields {
			if !requestFieldRequiresResourceID(field, c.idFieldAliases()) {
				continue
			}
			if _, ok := explicitRequestValue(values, field, preferredID); ok {
				return true
			}
		}
		return false
	}

	requestStruct, ok := operationRequestStruct(getOp.NewRequest)
	if !ok {
		return false
	}
	for i := 0; i < requestStruct.NumField(); i++ {
		fieldType, inspect := heuristicGetField(requestStruct, i)
		if !inspect {
			continue
		}
		if containsString(c.idFieldAliases(), requestLookupKey(fieldType)) {
			return true
		}
	}
	return false
}

func (c ServiceClient[T]) readResourceForExistingBeforeCreate(ctx context.Context, resource T) (any, bool, error) {
	state, err := c.prepareReadResourceState(resource, "")
	if err != nil {
		return nil, false, err
	}

	state, response, handled, trackedIDStale, err := c.readResourceWithGetForExistingBeforeCreate(ctx, resource, state)
	if handled {
		return response, trackedIDStale, err
	}

	response, err = c.readResourceWithList(ctx, resource, state, readPhaseCreate)
	return response, trackedIDStale, err
}

func (c ServiceClient[T]) readResourceWithGetForExistingBeforeCreate(ctx context.Context, resource T, state readResourceState) (readResourceState, any, bool, bool, error) {
	getOp := c.getReadOperation()
	if getOp == nil || !c.canInvokeGet(resource, state.readID) {
		return state, nil, false, false, nil
	}

	trackedIDStale := c.trackedStatusIDCanBeClearedAfterGetNotFound(resource, state.readID)
	response, err := c.invoke(ctx, getOp, resource, state.readID, requestBuildOptions{})
	if err == nil {
		return state, response, true, false, nil
	}
	if !isReadNotFound(err) || c.listReadOperation() == nil {
		return state, nil, true, false, err
	}

	return c.fallbackReadResourceState(resource, state, readPhaseCreate), nil, false, trackedIDStale, nil
}

func (c ServiceClient[T]) resolveExistingBeforeCreate(ctx context.Context, resource T, identity any) (any, bool, error) {
	if skipExistingBeforeCreate(ctx) {
		return nil, false, nil
	}
	if c.config.Create == nil {
		return nil, false, nil
	}

	trackedIDStale := false
	statePrepared := false
	var state readResourceState

	if c.currentID(resource) != "" {
		getOp := c.getReadOperation()
		if getOp == nil {
			return nil, false, nil
		}

		var err error
		state, err = c.prepareReadResourceState(resource, "")
		if err != nil {
			return nil, false, err
		}
		if !c.canInvokeGet(resource, state.readID) {
			return nil, false, nil
		}

		response, handled, getTrackedIDStale, err := func() (any, bool, bool, error) {
			nextState, nextResponse, handled, trackedIDStale, err := c.readResourceWithGetForExistingBeforeCreate(ctx, resource, state)
			state = nextState
			return nextResponse, handled, trackedIDStale, err
		}()
		trackedIDStale = getTrackedIDStale
		switch {
		case handled && err == nil:
			return response, false, nil
		case handled && !errors.Is(err, errResourceNotFound):
			return nil, trackedIDStale, err
		case !handled:
			statePrepared = true
		}
	}

	decision, err := c.guardExistingBeforeCreate(ctx, resource)
	if err != nil {
		if trackedIDStale {
			c.clearTrackedIdentity(resource)
		}
		return nil, trackedIDStale, err
	}
	if decision == ExistingBeforeCreateDecisionSkip {
		return nil, trackedIDStale, nil
	}

	if response, err := c.lookupExistingByIdentity(ctx, resource, identity); err != nil {
		return nil, trackedIDStale, err
	} else if response != nil {
		return response, trackedIDStale, nil
	}
	if !c.shouldResolveExistingBeforeCreate() {
		return nil, trackedIDStale, nil
	}

	if !statePrepared {
		state, err = c.prepareReadResourceState(resource, "")
		if err != nil {
			return nil, trackedIDStale, err
		}
	}

	response, err := c.readResourceWithList(ctx, resource, state, readPhaseCreate)
	if err == nil {
		return response, trackedIDStale, nil
	}
	if errors.Is(err, errResourceNotFound) {
		return nil, trackedIDStale, nil
	}
	return nil, trackedIDStale, err
}

func (c ServiceClient[T]) loadLiveMutationResponse(ctx context.Context, resource T, currentID string, existingResponse any, resolvedBeforeCreate bool) (string, any, error) {
	liveResponse := existingResponse
	if currentID == "" || !c.requiresLiveMutationAssessment() {
		c.mergeLiveResponseIntoStatus(resource, currentID, liveResponse)
		return currentID, liveResponse, nil
	}

	forceLiveGet := resolvedBeforeCreate && c.getReadOperation() != nil
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

func (c ServiceClient[T]) applyExistingResourceHooks(
	ctx context.Context,
	resource T,
	state createOrUpdateState,
	namespace string,
) (servicemanager.OSOKResponse, error, bool) {
	if state.currentID == "" || state.liveResponse == nil {
		return servicemanager.OSOKResponse{}, nil, false
	}
	if c.shouldObserveCurrentLifecycle(state.liveResponse) {
		if response, handled, err := c.applyStatusHooksObservation(resource, state.liveResponse, state.identity); handled {
			return response, err, true
		}
	}
	return c.handleParityHooks(ctx, resource, state, namespace)
}

func (c ServiceClient[T]) handleParityHooks(
	ctx context.Context,
	resource T,
	state createOrUpdateState,
	namespace string,
) (servicemanager.OSOKResponse, error, bool) {
	if c.config.ParityHooks.RequiresParityHandling == nil || !c.config.ParityHooks.RequiresParityHandling(resource, state.liveResponse) {
		return servicemanager.OSOKResponse{}, nil, false
	}

	shouldUpdate, err := c.shouldInvokeUpdate(ctx, resource, namespace, state.liveResponse)
	if err != nil {
		return servicemanager.OSOKResponse{}, err, true
	}
	if shouldUpdate {
		if c.config.ParityHooks.ApplyParityUpdate == nil {
			return servicemanager.OSOKResponse{}, fmt.Errorf("%s parity hooks require ApplyParityUpdate when RequiresParityHandling returns true", c.config.Kind), true
		}
		response, err := c.config.ParityHooks.ApplyParityUpdate(ctx, resource, state.liveResponse)
		return response, err, true
	}

	if response, handled, err := c.applyStatusHooksObservation(resource, state.liveResponse, state.identity); handled {
		return response, err, true
	}

	response, err := c.observeExistingResource(ctx, resource, state.currentID, state.liveResponse, state.identity)
	return response, err, true
}

func (c ServiceClient[T]) applyStatusHooksObservation(
	resource T,
	response any,
	identity any,
) (servicemanager.OSOKResponse, bool, error) {
	if c.config.StatusHooks.ProjectStatus == nil && c.config.StatusHooks.ApplyLifecycle == nil {
		return servicemanager.OSOKResponse{}, false, nil
	}
	if err := c.projectStatusWithHooks(resource, response); err != nil {
		return servicemanager.OSOKResponse{}, true, err
	}
	if responseID := responseID(response); responseID != "" && c.config.Identity.RecordTracked != nil {
		c.recordTrackedIdentity(resource, identity, responseID)
	}
	if c.config.StatusHooks.ApplyLifecycle != nil {
		projected, err := c.config.StatusHooks.ApplyLifecycle(resource, response)
		return projected, true, err
	}
	projected, err := c.applySuccessWithIdentity(resource, response, shared.Active, identity)
	return projected, true, err
}

func (c ServiceClient[T]) projectStatusWithHooks(resource T, response any) error {
	if c.config.StatusHooks.ProjectStatus != nil {
		return c.config.StatusHooks.ProjectStatus(resource, response)
	}
	return mergeResponseIntoStatus(resource, response)
}

func (c ServiceClient[T]) reconcileExistingResource(ctx context.Context, resource T, state createOrUpdateState, namespace string) (servicemanager.OSOKResponse, error) {
	shouldUpdate, err := c.shouldInvokeUpdate(ctx, resource, namespace, state.liveResponse)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	if shouldUpdate {
		return c.updateExistingResource(ctx, resource, state.currentID, namespace, state.liveResponse, state.identity)
	}
	return c.observeExistingResource(ctx, resource, state.currentID, state.liveResponse, state.identity)
}

func (c ServiceClient[T]) updateExistingResource(ctx context.Context, resource T, currentID string, namespace string, currentResponse any, identity any) (servicemanager.OSOKResponse, error) {
	options := c.requestBuildOptions(ctx, namespace)
	options.CurrentResponse = currentResponse

	response, err := c.invoke(ctx, c.config.Update, resource, currentID, options)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	c.seedOpeningRequestID(resource, response)
	if c.generatedWorkRequestPhaseEnabled(shared.OSOKAsyncPhaseUpdate) {
		workRequestID, err := c.startGeneratedWorkRequest(resource, response, shared.OSOKAsyncPhaseUpdate, identity)
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
		return c.resumeGeneratedWorkRequestCreateOrUpdate(ctx, resource, identity, workRequestID, shared.OSOKAsyncPhaseUpdate)
	}
	c.seedOpeningWorkRequestID(resource, response, shared.OSOKAsyncPhaseUpdate)

	response, err = c.followUpAfterWrite(ctx, resource, currentID, response, "update")
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	return c.applySuccessWithIdentity(resource, response, shared.Updating, identity)
}

func (c ServiceClient[T]) observeExistingResource(ctx context.Context, resource T, currentID string, liveResponse any, identity any) (servicemanager.OSOKResponse, error) {
	response := liveResponse
	if response == nil && c.hasReadableOperation() {
		var err error
		response, err = c.readResource(ctx, resource, currentID, readPhaseObserve)
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
	}
	return c.applySuccessWithIdentity(resource, response, shared.Active, identity)
}

func (c ServiceClient[T]) createOrReadResource(ctx context.Context, resource T, namespace string, identity any) (servicemanager.OSOKResponse, error) {
	if c.config.Create != nil {
		response, err := c.invoke(ctx, c.config.Create, resource, "", c.requestBuildOptions(ctx, namespace))
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
		c.seedOpeningRequestID(resource, response)
		if c.generatedWorkRequestPhaseEnabled(shared.OSOKAsyncPhaseCreate) {
			workRequestID, err := c.startGeneratedWorkRequest(resource, response, shared.OSOKAsyncPhaseCreate, identity)
			if err != nil {
				return c.failCreateOrUpdate(resource, err)
			}
			return c.resumeGeneratedWorkRequestCreateOrUpdate(ctx, resource, identity, workRequestID, shared.OSOKAsyncPhaseCreate)
		}
		c.seedOpeningWorkRequestID(resource, response, shared.OSOKAsyncPhaseCreate)

		followUp, err := c.followUpAfterWrite(ctx, resource, responseID(response), response, "create")
		if err != nil {
			return c.failCreateOrUpdate(resource, err)
		}
		return c.applySuccessWithIdentity(resource, followUp, shared.Provisioning, identity)
	}

	response, err := c.readResource(ctx, resource, "", readPhaseObserve)
	if err != nil {
		return c.failCreateOrUpdate(resource, err)
	}
	return c.applySuccessWithIdentity(resource, response, shared.Active, identity)
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
	identity, err := c.prepareIdentity(resource)
	if err != nil {
		return false, err
	}

	var restoreSyntheticTrackedID func()
	if c.currentID(resource) == "" && c.config.Identity.SeedSyntheticTrackedID != nil {
		restoreSyntheticTrackedID = c.config.Identity.SeedSyntheticTrackedID(resource, identity)
	}
	if workRequestID, phase := c.currentGeneratedWorkRequest(resource); workRequestID != "" && phase == shared.OSOKAsyncPhaseDelete {
		deleted, err := c.resumeGeneratedWorkRequestDelete(ctx, resource, workRequestID)
		if err != nil {
			if restoreSyntheticTrackedID != nil {
				restoreSyntheticTrackedID()
			}
			return false, err
		}
		c.recordTrackedIdentity(resource, identity, c.currentID(resource))
		if restoreSyntheticTrackedID != nil && c.config.Identity.RecordTracked == nil {
			restoreSyntheticTrackedID()
		}
		return deleted, nil
	}

	var deleted bool
	if c.config.Semantics != nil {
		deleted, err = c.deleteWithSemantics(ctx, resource)
	} else {
		deleted, err = c.deleteWithoutSemantics(ctx, resource)
	}
	if err != nil {
		if restoreSyntheticTrackedID != nil {
			restoreSyntheticTrackedID()
		}
		return false, err
	}

	c.recordTrackedIdentity(resource, identity, c.currentID(resource))
	if restoreSyntheticTrackedID != nil && c.config.Identity.RecordTracked == nil {
		restoreSyntheticTrackedID()
	}
	return deleted, nil
}

func (c ServiceClient[T]) validateDeleteRequest(resource T) error {
	if c.config.InitError != nil {
		return c.config.InitError
	}
	_, err := resourceStruct(resource)
	return err
}

func (c ServiceClient[T]) markDeletedWithHooks(resource T, message string) {
	if c.config.StatusHooks.MarkDeleted != nil {
		c.config.StatusHooks.MarkDeleted(resource, message)
		if status, err := osokStatus(resource); err == nil {
			c.clearAsyncOperation(status, resource)
		}
		return
	}
	c.markDeleted(resource, message)
}

func (c ServiceClient[T]) markTerminatingWithHooks(resource T, response any) error {
	if c.config.StatusHooks.MarkTerminating != nil {
		c.config.StatusHooks.MarkTerminating(resource, response)
		if c.generatedWorkRequestAsyncEnabled() {
			status, err := osokStatus(resource)
			if err != nil {
				return err
			}
			if status.Async.Current == nil || status.Async.Current.Source == shared.OSOKAsyncSourceWorkRequest {
				message := strings.TrimSpace(status.Message)
				if message == "" {
					message = "OCI resource delete is in progress"
				}
				now := metav1.Now()
				_ = c.applyAsyncOperation(status, resource, &shared.OSOKAsyncOperation{
					Source:          shared.OSOKAsyncSourceLifecycle,
					Phase:           shared.OSOKAsyncPhaseDelete,
					NormalizedClass: shared.OSOKAsyncClassPending,
					Message:         message,
					UpdatedAt:       &now,
				})
			}
		}
		return nil
	}
	c.markCondition(resource, shared.Terminating, "OCI resource delete is in progress")
	return nil
}

func (c ServiceClient[T]) invokeDeleteOperation(ctx context.Context, resource T, currentID string) (bool, error) {
	response, err := c.invoke(ctx, c.config.Delete, resource, currentID, requestBuildOptions{})
	if err != nil {
		err = c.handleDeleteError(resource, err)
		if isDeleteNotFound(err) {
			c.recordErrorRequestID(resource, err)
			c.markDeletedWithHooks(resource, "OCI resource no longer exists")
			return true, nil
		}
		return false, err
	}
	c.seedOpeningRequestID(resource, response)
	if c.generatedWorkRequestPhaseEnabled(shared.OSOKAsyncPhaseDelete) {
		_, err := c.startGeneratedWorkRequest(resource, response, shared.OSOKAsyncPhaseDelete, nil)
		return false, err
	}
	c.seedOpeningWorkRequestID(resource, response, shared.OSOKAsyncPhaseDelete)
	return false, nil
}

func (c ServiceClient[T]) deleteWithoutSemantics(ctx context.Context, resource T) (bool, error) {
	if c.config.Delete == nil {
		c.markDeletedWithHooks(resource, "OCI delete is not supported for this generated resource")
		return true, nil
	}

	currentID := c.currentID(resource)
	if currentID == "" {
		c.markDeletedWithHooks(resource, "OCI resource identifier is not recorded")
		return true, nil
	}
	if deleted, err := c.invokeDeleteOperation(ctx, resource, currentID); deleted || err != nil {
		return deleted, err
	}
	if workRequestID, phase := c.currentGeneratedWorkRequest(resource); workRequestID != "" && phase == shared.OSOKAsyncPhaseDelete {
		return c.resumeGeneratedWorkRequestDelete(ctx, resource, workRequestID)
	}
	return c.confirmDeleteWithoutSemantics(ctx, resource, currentID)
}

func (c ServiceClient[T]) confirmDeleteWithoutSemantics(ctx context.Context, resource T, currentID string) (bool, error) {
	if !c.hasDeleteConfirmRead() {
		c.markDeletedWithHooks(resource, "OCI delete request accepted")
		return true, nil
	}

	response, err := c.confirmDeleteRead(ctx, resource, currentID)
	if err != nil {
		if isDeleteNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.recordErrorRequestID(resource, err)
			c.markDeletedWithHooks(resource, "OCI resource deleted")
			return true, nil
		}
		return false, err
	}

	if err := c.projectStatusWithHooks(resource, response); err != nil {
		return false, err
	}
	outcome, err := c.applyDeleteOutcomeHooks(resource, response, DeleteConfirmStageAfterRequest)
	if err != nil {
		return false, err
	}
	if outcome.Handled {
		return outcome.Deleted, nil
	}
	if err := c.markTerminatingWithHooks(resource, response); err != nil {
		return false, err
	}
	return false, nil
}

func (c ServiceClient[T]) followUpAfterWrite(ctx context.Context, resource T, preferredID string, response any, phase string) (any, error) {
	if !c.requiresWriteFollowUp(phase) {
		return response, nil
	}
	if !c.hasReadableOperation() {
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
		return c.hasReadableOperation()
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
			c.markDeletedWithHooks(resource, "OCI resource no longer exists")
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
	if workRequestID, phase := c.currentGeneratedWorkRequest(resource); workRequestID != "" && phase == shared.OSOKAsyncPhaseDelete {
		return c.resumeGeneratedWorkRequestDelete(ctx, resource, workRequestID)
	}
	return c.confirmDeleteWithSemantics(ctx, resource, currentID, semantics)
}

func (c ServiceClient[T]) confirmDeleteIfAlreadyPending(ctx context.Context, resource T, currentID string, semantics *Semantics) (bool, error, bool) {
	if semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		return false, nil, false
	}
	if !c.hasDeleteConfirmRead() {
		return false, nil, false
	}

	response, err := c.confirmDeleteRead(ctx, resource, currentID)
	if err != nil {
		if isDeleteNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.markDeletedWithHooks(resource, "OCI resource deleted")
			return true, nil, true
		}
		return false, nil, false
	}

	if c.config.DeleteHooks.ApplyOutcome != nil {
		if err := c.projectStatusWithHooks(resource, response); err != nil {
			return false, err, true
		}
		outcome, err := c.applyDeleteOutcomeHooks(resource, response, DeleteConfirmStageAlreadyPending)
		if err != nil {
			return false, err, true
		}
		if outcome.Handled {
			return outcome.Deleted, nil, true
		}
	}

	lifecycleState := strings.ToUpper(responseLifecycleState(response))
	if !containsString(semantics.Delete.PendingStates, lifecycleState) &&
		!containsString(semantics.Delete.TerminalStates, lifecycleState) {
		return false, nil, false
	}

	if c.config.DeleteHooks.ApplyOutcome == nil {
		if err := c.projectStatusWithHooks(resource, response); err != nil {
			return false, err, true
		}
	}
	deleted, err := c.applyDeletePolicy(resource, response, semantics)
	return deleted, err, true
}

func (c ServiceClient[T]) shouldConfirmDeleteAfterError(err error) bool {
	if err == nil || !isRetryableDeleteConflict(err) {
		return false
	}
	if c.config.Semantics == nil || c.config.Semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		return false
	}
	return c.hasDeleteConfirmRead()
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
		c.markDeletedWithHooks(resource, "OCI delete request accepted")
		return true, nil
	}
	if !c.hasDeleteConfirmRead() {
		return false, fmt.Errorf("%s formal delete confirmation requires a readable OCI operation", c.config.Kind)
	}

	response, err := c.confirmDeleteRead(ctx, resource, currentID)
	if err != nil {
		if isDeleteNotFound(err) || errors.Is(err, errResourceNotFound) {
			c.recordErrorRequestID(resource, err)
			c.markDeletedWithHooks(resource, "OCI resource deleted")
			return true, nil
		}
		return false, err
	}

	if err := c.projectStatusWithHooks(resource, response); err != nil {
		return false, err
	}
	outcome, err := c.applyDeleteOutcomeHooks(resource, response, DeleteConfirmStageAfterRequest)
	if err != nil {
		return false, err
	}
	if outcome.Handled {
		return outcome.Deleted, nil
	}
	return c.applyDeletePolicy(resource, response, semantics)
}

func (c ServiceClient[T]) applyDeletePolicy(resource T, response any, semantics *Semantics) (bool, error) {
	lifecycleState := strings.ToUpper(responseLifecycleState(response))
	switch semantics.Delete.Policy {
	case "best-effort":
		return c.bestEffortDeleteOutcome(resource, response, lifecycleState, semantics)
	case "required":
		return c.requiredDeleteOutcome(resource, response, lifecycleState, semantics)
	default:
		return false, fmt.Errorf("%s formal delete confirmation policy %q is not supported", c.config.Kind, semantics.Delete.Policy)
	}
}

func (c ServiceClient[T]) bestEffortDeleteOutcome(resource T, response any, lifecycleState string, semantics *Semantics) (bool, error) {
	if lifecycleState == "" ||
		containsString(semantics.Delete.PendingStates, lifecycleState) ||
		containsString(semantics.Delete.TerminalStates, lifecycleState) {
		c.markDeletedWithHooks(resource, "OCI delete request accepted")
		return true, nil
	}

	if err := c.markTerminatingWithHooks(resource, response); err != nil {
		return false, err
	}
	return false, nil
}

func (c ServiceClient[T]) requiredDeleteOutcome(resource T, response any, lifecycleState string, semantics *Semantics) (bool, error) {
	switch {
	case containsString(semantics.Delete.TerminalStates, lifecycleState):
		c.markDeletedWithHooks(resource, "OCI resource deleted")
		return true, nil
	case lifecycleState == "" || containsString(semantics.Delete.PendingStates, lifecycleState):
		if err := c.markTerminatingWithHooks(resource, response); err != nil {
			return false, err
		}
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

	if !c.hasDeleteConfirmRead() {
		return "", errResourceNotFound
	}

	response, err := c.confirmDeleteRead(ctx, resource, "")
	if err != nil {
		return "", err
	}
	currentID = responseID(response)
	if currentID == "" {
		return "", fmt.Errorf("%s delete confirmation could not resolve a resource OCID", c.config.Kind)
	}
	if err := c.projectStatusWithHooks(resource, response); err != nil {
		return "", err
	}
	return currentID, nil
}

func skipExistingBeforeCreate(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	skip, _ := ctx.Value(skipExistingBeforeCreateContextKey).(bool)
	return skip
}

func (c ServiceClient[T]) shouldResolveExistingBeforeCreate() bool {
	return c.config.Create != nil && c.listReadOperation() != nil && c.config.Semantics != nil && c.config.Semantics.List != nil
}

func (c ServiceClient[T]) requiresLiveMutationAssessment() bool {
	return c.config.Semantics != nil &&
		(len(c.config.Semantics.Mutation.ForceNew) > 0 || len(c.config.Semantics.Mutation.Mutable) > 0) &&
		c.hasReadableOperation()
}

func (c ServiceClient[T]) shouldInvokeUpdate(ctx context.Context, resource T, namespace string, currentResponse any) (bool, error) {
	if c.config.Update == nil {
		return false, nil
	}
	if c.shouldObserveCurrentLifecycle(currentResponse) {
		return false, nil
	}
	if c.config.BuildUpdateBody != nil {
		_, updateNeeded, err := c.config.BuildUpdateBody(ctx, resource, namespace, currentResponse)
		return updateNeeded, err
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
	if err := c.validateForceNewFields(resource, specValues, currentValues); err != nil {
		return err
	}
	if err := c.validateCreateOnlyDrift(resource, currentResponse); err != nil {
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

func (c ServiceClient[T]) validateCreateOnlyDrift(resource T, currentResponse any) error {
	if currentResponse == nil || c.config.ParityHooks.ValidateCreateOnlyDrift == nil {
		return nil
	}
	return c.config.ParityHooks.ValidateCreateOnlyDrift(resource, currentResponse)
}

func mutationValues(resource any, currentResponse any) (map[string]any, map[string]any, error) {
	resourceValue, err := resourceStruct(resource)
	if err != nil {
		return nil, nil, err
	}

	specValues := jsonMap(fieldInterface(resourceValue, "Spec"))
	currentValues := jsonMap(fieldInterface(resourceValue, "Status"))
	if specValues == nil {
		specValues = map[string]any{}
	}
	if currentValues == nil {
		currentValues = map[string]any{}
	}
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

func (c ServiceClient[T]) validateForceNewFields(resource T, specValues map[string]any, currentValues map[string]any) error {
	for _, field := range c.config.Semantics.Mutation.ForceNew {
		wantedValue, specOK := lookupMeaningfulValue(specValues, field)
		if !specOK {
			var err error
			wantedValue, specOK, err = meaningfulMutationValueByPath(specValue(resource), field)
			if err != nil {
				return err
			}
		}
		statusValue, statusOK := lookupValueByPath(currentValues, field)
		if !specOK || !statusOK {
			continue
		}
		if !forceNewValuesEqual(wantedValue, statusValue) {
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
		wantedValue, specFound := lookupMeaningfulValue(specValues, field)
		if !specFound {
			wantedValue, specFound, err = meaningfulMutationValueByPath(fieldInterface(resourceValue, "Spec"), field)
			if err != nil {
				return false, err
			}
		}
		if !specFound {
			continue
		}
		currentValue, currentFound := lookupMeaningfulValue(currentValues, field)
		if !currentFound {
			if responseExposesFieldPath(currentResponse, field) {
				return true, nil
			}
			continue
		}
		if !valuesEqual(wantedValue, currentValue) {
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
	getOp := c.getReadOperation()
	if getOp == nil {
		return nil, fmt.Errorf("%s generated runtime has no OCI Get operation for live mutation validation", c.config.Kind)
	}

	response, err := c.invoke(ctx, getOp, resource, currentID, requestBuildOptions{})
	if err != nil {
		if isReadNotFound(err) {
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
	getOp := c.getReadOperation()
	if getOp == nil || !c.canInvokeGet(resource, state.readID) {
		return state, nil, false, nil
	}

	response, err := c.invoke(ctx, getOp, resource, state.readID, requestBuildOptions{})
	if err == nil {
		return state, response, true, nil
	}
	if !isReadNotFound(err) || c.listReadOperation() == nil {
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
	listOp := c.listReadOperation()
	if listOp == nil {
		return nil, fmt.Errorf("%s generated runtime has no readable OCI operation", c.config.Kind)
	}

	response, err := c.invokeWithValues(ctx, listOp, resource, state.listValues, state.listID, requestBuildOptions{})
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
	getOp := c.getReadOperation()
	if getOp == nil {
		return false
	}

	for _, field := range getOp.Fields {
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
	getOp := c.getReadOperation()
	if getOp == nil {
		return false
	}

	requestStruct, ok := operationRequestStruct(getOp.NewRequest)
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
	getOp := c.getReadOperation()
	if getOp == nil {
		return false
	}

	values, err := lookupValues(resource)
	if err != nil {
		return true
	}

	if len(getOp.Fields) > 0 {
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
		if c.config.BuildUpdateBody != nil {
			body, ok, err := c.config.BuildUpdateBody(options.Context, resource, options.Namespace, options.CurrentResponse)
			if err != nil {
				return nil, false, fmt.Errorf("build %s update body: %w", c.config.Kind, err)
			}
			if ok {
				return body, true, nil
			}
			return nil, false, nil
		}
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

	resolvedSpec, err := resolvedSpecValueWithDecoder(resource, options, decodedJSONValueWithBoolFields, true)
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
			currentValues, err = mutationJSONMap(body)
			if err != nil {
				return nil, false, err
			}
		}
	}
	if len(currentValues) == 0 {
		statusValue, err := statusStruct(resource)
		if err != nil {
			return nil, false, err
		}
		currentValues, err = mutationJSONMap(statusValue.Interface())
		if err != nil {
			return nil, false, err
		}
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
	return c.applySuccessWithIdentity(resource, response, fallback, nil)
}

func (c ServiceClient[T]) applySuccessWithIdentity(resource T, response any, fallback shared.OSOKConditionType, identity any) (servicemanager.OSOKResponse, error) {
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
	if c.config.Identity.RecordTracked != nil {
		c.recordTrackedIdentity(resource, identity, resourceID)
	} else if resourceID != "" {
		status.Ocid = shared.OCID(resourceID)
	}

	now := metav1.Now()
	if resourceID != "" && status.CreatedAt == nil {
		status.CreatedAt = &now
	}
	evaluation := c.classifyLifecycleAsync(response, status, fallback)
	if evaluation.current != nil {
		evaluation.current.UpdatedAt = &now
		projection := c.applyAsyncOperation(status, resource, evaluation.current)
		return servicemanager.OSOKResponse{
			IsSuccessful:    projection.Condition != shared.Failed,
			ShouldRequeue:   projection.ShouldRequeue,
			RequeueDuration: defaultRequeueDuration,
		}, nil
	}

	c.clearAsyncOperation(status, resource)
	status.Message = evaluation.message
	status.Reason = string(evaluation.condition)
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(*status, evaluation.condition, conditionStatusForCondition(evaluation.condition), "", evaluation.message, c.config.Log)

	return servicemanager.OSOKResponse{
		IsSuccessful:    evaluation.condition != shared.Failed,
		ShouldRequeue:   evaluation.shouldRequeue,
		RequeueDuration: defaultRequeueDuration,
	}, nil
}

func (c ServiceClient[T]) markFailure(resource T, err error) error {
	status, statusErr := osokStatus(resource)
	if statusErr != nil {
		return err
	}
	servicemanager.RecordErrorOpcRequestID(status, err)
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	now := metav1.Now()
	status.UpdatedAt = &now
	if status.Async.Current != nil {
		current := *status.Async.Current
		current.NormalizedClass = shared.OSOKAsyncClassFailed
		current.Message = err.Error()
		current.UpdatedAt = &now
		_ = c.applyAsyncOperation(status, resource, &current)
		return err
	}
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
	c.clearAsyncOperation(status, resource)
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
	if condition == shared.Terminating {
		current := &shared.OSOKAsyncOperation{
			Source:          shared.OSOKAsyncSourceLifecycle,
			Phase:           shared.OSOKAsyncPhaseDelete,
			NormalizedClass: shared.OSOKAsyncClassPending,
			Message:         message,
			UpdatedAt:       &now,
		}
		_ = c.applyAsyncOperation(status, resource, current)
		return
	}
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

// ResolveSpecValueWithBoolFields rewrites secret-backed spec inputs while
// preserving explicit false booleans for request-body projection.
func ResolveSpecValueWithBoolFields(resource any, ctx context.Context, credentialClient credhelper.CredentialClient, namespace string) (any, error) {
	return resolvedSpecValueWithDecoder(resource, requestBuildOptions{
		Context:          ctx,
		CredentialClient: credentialClient,
		Namespace:        namespace,
	}, decodedJSONValueWithBoolFields, true)
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
	specValue := fieldInterface(resourceValue, "Spec")
	if specRoot := jsonMap(specValue); specRoot != nil {
		values[lookupSpecRootKey] = specRoot
	}
	mergeJSONMap(values, specValue)
	statusValue := fieldInterface(resourceValue, "Status")
	if statusRoot := jsonMap(statusValue); statusRoot != nil {
		values[lookupStatusRootKey] = statusRoot
	}
	mergeJSONMap(values, statusValue)
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
	return resolvedSpecValueWithDecoder(resource, options, decodedJSONValue, false)
}

func resolvedSpecValueWithDecoder(resource any, options requestBuildOptions, decoder func(any) (any, error), preserveZeroStructDecoded bool) (any, error) {
	raw := specValue(resource)
	if raw == nil {
		return nil, nil
	}

	decoded, err := decoder(raw)
	if err != nil {
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

	resolved, _, err := rewriteSecretSources(specField, decoded, options, preserveZeroStructDecoded)
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

func rewriteSecretSources(value reflect.Value, decoded any, options requestBuildOptions, preserveZeroStructDecoded bool) (any, bool, error) {
	value, ok := indirectValue(value)
	if !ok {
		return nil, false, nil
	}
	if rewritten, include, handled, err := rewriteSharedSecretSource(value, options); handled {
		return rewritten, include, err
	}
	if value.Kind() == reflect.Struct && value.IsZero() {
		if !preserveZeroStructDecoded {
			return nil, false, nil
		}
		if decodedMap, ok := decoded.(map[string]any); !ok || !meaningfulValue(decodedMap) {
			return nil, false, nil
		}
	}

	switch value.Kind() {
	case reflect.Struct:
		return rewriteSecretStruct(value, decoded, options, preserveZeroStructDecoded)
	case reflect.Slice, reflect.Array:
		return rewriteSecretSlice(value, decoded, options, preserveZeroStructDecoded)
	case reflect.Map:
		return rewriteSecretMap(value, decoded, options, preserveZeroStructDecoded)
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

func rewriteSecretStruct(value reflect.Value, decoded any, options requestBuildOptions, preserveZeroStructDecoded bool) (any, bool, error) {
	decodedMap := decodedMapValue(decoded)
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() {
			continue
		}
		if fieldType.Anonymous && embeddedJSONField(fieldType) {
			var err error
			decodedMap, err = rewriteEmbeddedSecretField(value.Field(i), decodedMap, options, preserveZeroStructDecoded)
			if err != nil {
				return nil, false, err
			}
			continue
		}
		if err := rewriteNamedSecretField(decodedMap, value.Field(i), fieldType, options, preserveZeroStructDecoded); err != nil {
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

func rewriteEmbeddedSecretField(fieldValue reflect.Value, decodedMap map[string]any, options requestBuildOptions, preserveZeroStructDecoded bool) (map[string]any, error) {
	rewritten, _, err := rewriteSecretSources(fieldValue, decodedMap, options, preserveZeroStructDecoded)
	if err != nil {
		return nil, err
	}
	if nestedMap, ok := rewritten.(map[string]any); ok {
		return nestedMap, nil
	}
	return decodedMap, nil
}

func rewriteNamedSecretField(decodedMap map[string]any, fieldValue reflect.Value, fieldType reflect.StructField, options requestBuildOptions, preserveZeroStructDecoded bool) error {
	jsonName, skip := fieldJSONTagName(fieldType)
	if skip {
		return nil
	}
	childDecoded, exists := decodedMap[jsonName]
	rewritten, include, err := rewriteSecretSources(fieldValue, childDecoded, options, preserveZeroStructDecoded)
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

func rewriteSecretSlice(value reflect.Value, decoded any, options requestBuildOptions, preserveZeroStructDecoded bool) (any, bool, error) {
	decodedSlice, ok := decoded.([]any)
	if !ok {
		return decoded, true, nil
	}
	for i := 0; i < value.Len() && i < len(decodedSlice); i++ {
		rewritten, include, err := rewriteSecretSources(value.Index(i), decodedSlice[i], options, preserveZeroStructDecoded)
		if err != nil {
			return nil, false, err
		}
		if include {
			decodedSlice[i] = rewritten
		}
	}
	return decodedSlice, true, nil
}

func rewriteSecretMap(value reflect.Value, decoded any, options requestBuildOptions, preserveZeroStructDecoded bool) (any, bool, error) {
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
		rewritten, include, err := rewriteSecretSources(iter.Value(), childDecoded, options, preserveZeroStructDecoded)
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

func responseWorkRequestID(response any) string {
	if response == nil {
		return ""
	}

	value, ok := indirectValue(reflect.ValueOf(response))
	if !ok || value.Kind() != reflect.Struct {
		return ""
	}

	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() || !isWorkRequestHeaderField(fieldType) {
			continue
		}
		if workRequestID := stringFieldValue(value.Field(i)); workRequestID != "" {
			return workRequestID
		}
	}
	return ""
}

func responseRequestID(response any) string {
	return servicemanager.ResponseOpcRequestID(response)
}

func isWorkRequestHeaderField(fieldType reflect.StructField) bool {
	return fieldType.Name == "OpcWorkRequestId" ||
		(fieldType.Tag.Get("presentIn") == "header" && fieldType.Tag.Get("name") == "opc-work-request-id")
}

func stringFieldValue(value reflect.Value) string {
	value, ok := indirectValue(value)
	if !ok || value.Kind() != reflect.String {
		return ""
	}
	return strings.TrimSpace(value.String())
}

func workRequestStringField(workRequest any, fieldName string) string {
	return stringFieldValue(workRequestFieldValue(workRequest, fieldName))
}

func workRequestFloat32Field(workRequest any, fieldName string) *float32 {
	value, ok := indirectValue(workRequestFieldValue(workRequest, fieldName))
	if !ok {
		return nil
	}

	switch value.Kind() {
	case reflect.Float32, reflect.Float64:
		percent := float32(value.Convert(reflect.TypeOf(float32(0))).Float())
		return &percent
	default:
		return nil
	}
}

func workRequestFieldValue(workRequest any, fieldName string) reflect.Value {
	if strings.TrimSpace(fieldName) == "" {
		return reflect.Value{}
	}

	value, ok := indirectValue(reflect.ValueOf(workRequest))
	if !ok || value.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	return value.FieldByName(fieldName)
}

func (c ServiceClient[T]) seedOpeningWorkRequestID(resource T, response any, phase shared.OSOKAsyncPhase) {
	workRequestID := responseWorkRequestID(response)
	if workRequestID == "" || phase == "" {
		return
	}

	status, err := osokStatus(resource)
	if err != nil {
		return
	}

	now := metav1.Now()
	_ = c.applyAsyncOperation(status, resource, &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceWorkRequest,
		Phase:           phase,
		WorkRequestID:   workRequestID,
		NormalizedClass: shared.OSOKAsyncClassPending,
		UpdatedAt:       &now,
	})
}

func (c ServiceClient[T]) seedOpeningRequestID(resource T, response any) {
	status, err := osokStatus(resource)
	if err != nil {
		return
	}

	servicemanager.RecordResponseOpcRequestID(status, response)
}

func (c ServiceClient[T]) recordErrorRequestID(resource T, err error) {
	status, statusErr := osokStatus(resource)
	if statusErr != nil {
		return
	}

	servicemanager.RecordErrorOpcRequestID(status, err)
}

type lifecycleAsyncEvaluation struct {
	current       *shared.OSOKAsyncOperation
	condition     shared.OSOKConditionType
	shouldRequeue bool
	message       string
}

func (c ServiceClient[T]) classifyLifecycleAsync(response any, status *shared.OSOKStatus, fallback shared.OSOKConditionType) lifecycleAsyncEvaluation {
	if c.config.Semantics == nil {
		return classifyLifecycleAsyncHeuristics(response, status, fallback)
	}
	return classifyLifecycleAsyncSemantics(response, status, fallback, c.config.Semantics)
}

func classifyLifecycleSemantics(response any, fallback shared.OSOKConditionType, semantics *Semantics) (shared.OSOKConditionType, bool, string) {
	evaluation := classifyLifecycleAsyncSemantics(response, nil, fallback, semantics)
	return evaluation.condition, evaluation.shouldRequeue, evaluation.message
}

func classifyLifecycleAsyncSemantics(response any, status *shared.OSOKStatus, fallback shared.OSOKConditionType, semantics *Semantics) lifecycleAsyncEvaluation {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return lifecycleAsyncEvaluation{
			condition:     fallback,
			shouldRequeue: shouldRequeueForCondition(fallback),
			message:       defaultConditionMessage(fallback),
		}
	}

	values := jsonMap(body)
	lifecycleState := strings.ToUpper(firstNonEmpty(values, "lifecycleState", "status"))
	message := firstNonEmpty(values, "lifecycleDetails", "message", "displayName", "name")
	if message == "" {
		message = defaultConditionMessage(fallback)
	}

	switch {
	case lifecycleState == "":
		return lifecycleAsyncEvaluation{
			condition:     fallback,
			shouldRequeue: shouldRequeueForCondition(fallback),
			message:       message,
		}
	case containsString(semantics.Lifecycle.ProvisioningStates, lifecycleState):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending)
	case containsString(semantics.Lifecycle.UpdatingStates, lifecycleState):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending)
	case containsString(semantics.Delete.PendingStates, lifecycleState):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
	case containsString(semantics.Delete.TerminalStates, lifecycleState):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassSucceeded)
	case containsString(semantics.Lifecycle.ActiveStates, lifecycleState):
		return lifecycleAsyncEvaluation{condition: shared.Active, shouldRequeue: false, message: message}
	default:
		failureMessage := fmt.Sprintf("formal lifecycle state %q is not modeled: %s", lifecycleState, message)
		return lifecycleFailureEvaluation(status, fallback, lifecycleState, failureMessage, shared.OSOKAsyncClassUnknown)
	}
}

func classifyLifecycleHeuristics(response any, fallback shared.OSOKConditionType) (shared.OSOKConditionType, bool, string) {
	evaluation := classifyLifecycleAsyncHeuristics(response, nil, fallback)
	return evaluation.condition, evaluation.shouldRequeue, evaluation.message
}

func classifyLifecycleAsyncHeuristics(response any, status *shared.OSOKStatus, fallback shared.OSOKConditionType) lifecycleAsyncEvaluation {
	body, ok := responseBody(response)
	if !ok || body == nil {
		return lifecycleAsyncEvaluation{
			condition:     fallback,
			shouldRequeue: shouldRequeueForCondition(fallback),
			message:       defaultConditionMessage(fallback),
		}
	}

	values := jsonMap(body)
	lifecycleState := strings.ToUpper(firstNonEmpty(values, "lifecycleState", "status"))
	message := firstNonEmpty(values, "lifecycleDetails", "message", "displayName", "name")
	if message == "" {
		message = defaultConditionMessage(fallback)
	}

	switch {
	case lifecycleState == "":
		return lifecycleAsyncEvaluation{
			condition:     fallback,
			shouldRequeue: shouldRequeueForCondition(fallback),
			message:       message,
		}
	case strings.Contains(lifecycleState, "FAIL"),
		strings.Contains(lifecycleState, "ERROR"),
		strings.Contains(lifecycleState, "INOPERABLE"):
		return lifecycleFailureEvaluation(status, fallback, lifecycleState, message, shared.OSOKAsyncClassFailed)
	case strings.Contains(lifecycleState, "NEEDS_ATTENTION"):
		return lifecycleFailureEvaluation(status, fallback, lifecycleState, message, shared.OSOKAsyncClassAttention)
	case strings.Contains(lifecycleState, "DELETED"),
		strings.Contains(lifecycleState, "TERMINATED"):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassSucceeded)
	case strings.Contains(lifecycleState, "DELETE"),
		strings.Contains(lifecycleState, "TERMINAT"):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseDelete, shared.OSOKAsyncClassPending)
	case strings.Contains(lifecycleState, "UPDAT"),
		strings.Contains(lifecycleState, "MODIFY"),
		strings.Contains(lifecycleState, "PATCH"):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseUpdate, shared.OSOKAsyncClassPending)
	case strings.Contains(lifecycleState, "CREATE"),
		strings.Contains(lifecycleState, "PROVISION"),
		strings.Contains(lifecycleState, "PENDING"),
		strings.Contains(lifecycleState, "IN_PROGRESS"),
		strings.Contains(lifecycleState, "ACCEPT"),
		strings.Contains(lifecycleState, "START"):
		return newLifecycleAsyncEvaluation(status, message, lifecycleState, shared.OSOKAsyncPhaseCreate, shared.OSOKAsyncClassPending)
	default:
		return lifecycleAsyncEvaluation{condition: shared.Active, shouldRequeue: false, message: message}
	}
}

func newLifecycleAsyncEvaluation(status *shared.OSOKStatus, message string, lifecycleState string, phase shared.OSOKAsyncPhase, class shared.OSOKAsyncNormalizedClass) lifecycleAsyncEvaluation {
	if phase == "" {
		return lifecycleAsyncEvaluation{condition: shared.Failed, shouldRequeue: false, message: message}
	}

	current := &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           servicemanager.ResolveAsyncPhase(status, phase),
		RawStatus:       lifecycleState,
		NormalizedClass: class,
		Message:         message,
	}
	projection := servicemanager.ProjectAsyncCondition(class, current.Phase)
	if strings.TrimSpace(message) == "" {
		message = projection.DefaultMessage
		current.Message = message
	}

	return lifecycleAsyncEvaluation{
		current:       current,
		condition:     projection.Condition,
		shouldRequeue: projection.ShouldRequeue,
		message:       message,
	}
}

func lifecycleFailureEvaluation(status *shared.OSOKStatus, fallback shared.OSOKConditionType, lifecycleState string, message string, class shared.OSOKAsyncNormalizedClass) lifecycleAsyncEvaluation {
	phase := servicemanager.ResolveAsyncPhase(status, fallbackAsyncPhase(fallback))
	if phase == "" {
		return lifecycleAsyncEvaluation{condition: shared.Failed, shouldRequeue: false, message: message}
	}
	return newLifecycleAsyncEvaluation(status, message, lifecycleState, phase, class)
}

func fallbackAsyncPhase(condition shared.OSOKConditionType) shared.OSOKAsyncPhase {
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

func fallbackConditionForAsyncPhase(phase shared.OSOKAsyncPhase) shared.OSOKConditionType {
	switch phase {
	case shared.OSOKAsyncPhaseCreate:
		return shared.Provisioning
	case shared.OSOKAsyncPhaseUpdate:
		return shared.Updating
	case shared.OSOKAsyncPhaseDelete:
		return shared.Terminating
	default:
		return shared.Active
	}
}

func shouldRequeueForCondition(condition shared.OSOKConditionType) bool {
	return condition == shared.Provisioning || condition == shared.Updating || condition == shared.Terminating
}

func conditionStatusForCondition(condition shared.OSOKConditionType) v1.ConditionStatus {
	if condition == shared.Failed {
		return v1.ConditionFalse
	}
	return v1.ConditionTrue
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

func validateGeneratedWorkRequestAsyncHooks[T any](cfg Config[T]) error {
	if cfg.Semantics == nil || cfg.Semantics.Async == nil {
		return nil
	}

	async := cfg.Semantics.Async
	if strings.TrimSpace(async.Strategy) != asyncStrategyWorkRequest ||
		strings.TrimSpace(async.Runtime) != asyncRuntimeGeneratedRuntime {
		return nil
	}

	var problems []string
	if cfg.Async.GetWorkRequest == nil {
		problems = append(problems, "workrequest async semantics require Async.GetWorkRequest")
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("%s workrequest async hooks blocked: %s", cfg.Kind, strings.Join(problems, "; "))
}

func validateFormalSemantics(kind string, semantics *Semantics) error {
	if semantics == nil {
		return nil
	}

	problems := append([]string{}, unsupportedFormalProblems(semantics)...)
	problems = append(problems, unsupportedAuxiliaryProblems(semantics)...)
	problems = append(problems, unsupportedFollowUpHelpers(semantics)...)
	problems = append(problems, invalidAsyncSemanticsProblems(semantics)...)
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

func invalidAsyncSemanticsProblems(semantics *Semantics) []string {
	async := semantics.Async
	if async == nil {
		if hasWorkRequestHelper(semantics) {
			return []string{fmt.Sprintf("workrequest helper requires explicit async strategy %q", asyncStrategyWorkRequest)}
		}
		return nil
	}

	var problems []string
	strategy := strings.TrimSpace(async.Strategy)
	runtime := strings.TrimSpace(async.Runtime)
	formalClassification := strings.TrimSpace(async.FormalClassification)

	problems = append(problems, invalidAsyncStrategyProblems("async.strategy", strategy)...)
	problems = append(problems, invalidAsyncRuntimeProblems("async.runtime", runtime)...)
	problems = append(problems, invalidAsyncStrategyProblems("async.formalClassification", formalClassification)...)
	problems = append(problems, invalidAsyncWorkRequestProblems(async.WorkRequest)...)

	if !hasExplicitAsyncContract(async) {
		if hasWorkRequestHelper(semantics) {
			problems = append(problems, fmt.Sprintf("workrequest helper requires explicit async strategy %q", asyncStrategyWorkRequest))
		}
		return problems
	}

	if strategy == "" {
		problems = append(problems, "explicit async semantics require strategy")
	}
	if runtime == "" {
		problems = append(problems, "explicit async semantics require runtime")
	}
	if runtime == asyncRuntimeHandwritten {
		problems = append(problems, fmt.Sprintf("generatedruntime cannot honor explicit async runtime %q", runtime))
	}

	workRequestConfigured := hasWorkRequestSemantics(async.WorkRequest)
	switch strategy {
	case asyncStrategyWorkRequest:
		if !workRequestConfigured {
			problems = append(problems, "workrequest async semantics require workRequest metadata")
		} else {
			if strings.TrimSpace(async.WorkRequest.Source) == "" {
				problems = append(problems, "workrequest async semantics require workRequest.source")
			}
			if len(async.WorkRequest.Phases) == 0 {
				problems = append(problems, "workrequest async semantics require workRequest.phases")
			}
		}
	case "", asyncStrategyLifecycle, asyncStrategyNone:
		if workRequestConfigured {
			problems = append(problems, fmt.Sprintf("workRequest metadata requires strategy %q", asyncStrategyWorkRequest))
		}
	default:
		if workRequestConfigured {
			problems = append(problems, fmt.Sprintf("workRequest metadata requires strategy %q", asyncStrategyWorkRequest))
		}
	}
	if strategy != asyncStrategyWorkRequest && hasWorkRequestHelper(semantics) {
		problems = append(problems, fmt.Sprintf("workrequest helper requires explicit async strategy %q", asyncStrategyWorkRequest))
	}

	return problems
}

func invalidAsyncStrategyProblems(field string, strategy string) []string {
	switch strategy {
	case "", asyncStrategyLifecycle, asyncStrategyWorkRequest, asyncStrategyNone:
		return nil
	default:
		return []string{fmt.Sprintf(
			`%s %q must be one of %q, %q, or %q`,
			field,
			strategy,
			asyncStrategyLifecycle,
			asyncStrategyWorkRequest,
			asyncStrategyNone,
		)}
	}
}

func invalidAsyncRuntimeProblems(field string, runtime string) []string {
	switch runtime {
	case "", asyncRuntimeGeneratedRuntime, asyncRuntimeHandwritten:
		return nil
	default:
		return []string{fmt.Sprintf(
			`%s %q must be one of %q or %q`,
			field,
			runtime,
			asyncRuntimeGeneratedRuntime,
			asyncRuntimeHandwritten,
		)}
	}
}

func invalidAsyncWorkRequestProblems(workRequest *WorkRequestSemantics) []string {
	if workRequest == nil {
		return nil
	}

	var problems []string
	if source := strings.TrimSpace(workRequest.Source); source != "" {
		switch source {
		case asyncWorkRequestSourceServiceSDK, asyncWorkRequestSourceWorkRequestsAPI, asyncWorkRequestSourceProviderHelper:
		default:
			problems = append(problems, fmt.Sprintf(
				`async.workRequest.source %q must be one of %q, %q, or %q`,
				workRequest.Source,
				asyncWorkRequestSourceServiceSDK,
				asyncWorkRequestSourceWorkRequestsAPI,
				asyncWorkRequestSourceProviderHelper,
			))
		}
	}

	seen := make(map[string]struct{}, len(workRequest.Phases))
	for index, rawPhase := range workRequest.Phases {
		phase := strings.TrimSpace(rawPhase)
		switch phase {
		case asyncPhaseCreate, asyncPhaseUpdate, asyncPhaseDelete:
		case "":
			problems = append(problems, fmt.Sprintf("async.workRequest.phases[%d] must not be blank", index))
			continue
		default:
			problems = append(problems, fmt.Sprintf(
				`async.workRequest.phases[%d] %q must be one of %q, %q, or %q`,
				index,
				rawPhase,
				asyncPhaseCreate,
				asyncPhaseUpdate,
				asyncPhaseDelete,
			))
			continue
		}
		if _, exists := seen[phase]; exists {
			problems = append(problems, fmt.Sprintf(`async.workRequest.phases contains duplicate phase %q`, phase))
			continue
		}
		seen[phase] = struct{}{}
	}

	if workRequest.LegacyFieldBridge == nil {
		return problems
	}

	for subfield, value := range map[string]string{
		"create": workRequest.LegacyFieldBridge.Create,
		"update": workRequest.LegacyFieldBridge.Update,
		"delete": workRequest.LegacyFieldBridge.Delete,
	} {
		if value == "" {
			continue
		}
		if strings.TrimSpace(value) == "" {
			problems = append(problems, fmt.Sprintf("async.workRequest.legacyFieldBridge.%s must not be blank", subfield))
		}
	}

	return problems
}

func hasExplicitAsyncContract(async *AsyncSemantics) bool {
	if async == nil {
		return false
	}
	return strings.TrimSpace(async.Strategy) != "" ||
		strings.TrimSpace(async.Runtime) != "" ||
		strings.TrimSpace(async.FormalClassification) != "" ||
		hasWorkRequestSemantics(async.WorkRequest)
}

func hasWorkRequestSemantics(workRequest *WorkRequestSemantics) bool {
	if workRequest == nil {
		return false
	}
	if strings.TrimSpace(workRequest.Source) != "" || len(workRequest.Phases) > 0 {
		return true
	}
	if workRequest.LegacyFieldBridge == nil {
		return false
	}
	return strings.TrimSpace(workRequest.LegacyFieldBridge.Create) != "" ||
		strings.TrimSpace(workRequest.LegacyFieldBridge.Update) != "" ||
		strings.TrimSpace(workRequest.LegacyFieldBridge.Delete) != ""
}

func hasWorkRequestHelper(semantics *Semantics) bool {
	for _, hooks := range [][]Hook{
		semantics.Hooks.Create,
		semantics.Hooks.Update,
		semantics.Hooks.Delete,
		semantics.CreateFollowUp.Hooks,
		semantics.UpdateFollowUp.Hooks,
		semantics.DeleteFollowUp.Hooks,
	} {
		for _, hook := range hooks {
			if strings.TrimSpace(hook.Helper) == "tfresource.WaitForWorkRequestWithErrorHandling" {
				return true
			}
		}
	}
	return false
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

func isReadNotFound(err error) bool {
	if errors.Is(err, errResourceNotFound) {
		return true
	}
	return errorutil.ClassifyDeleteError(err).IsUnambiguousNotFound()
}

func isDeleteNotFound(err error) bool {
	if errors.Is(err, errResourceNotFound) {
		return true
	}
	classification := errorutil.ClassifyDeleteError(err)
	return classification.IsUnambiguousNotFound() || classification.IsAuthShapedNotFound()
}

func isRetryableDeleteConflict(err error) bool {
	classification := errorutil.ClassifyDeleteError(err)
	if classification.HTTPStatusCode != 409 {
		return false
	}

	switch classification.ErrorCode {
	case errorutil.IncorrectState, "ExternalServerIncorrectState":
		return true
	default:
		return false
	}
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
	case databaseToolsConnectionCreateDetailsType:
		body, err := convertDatabaseToolsConnectionCreateDetails(payload)
		if err != nil {
			return reflect.Value{}, true, err
		}
		converted := reflect.New(targetType).Elem()
		converted.Set(reflect.ValueOf(body))
		return converted, true, nil
	case databaseToolsConnectionUpdateDetailsType:
		body, err := convertDatabaseToolsConnectionUpdateDetails(payload)
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

func convertDatabaseToolsConnectionCreateDetails(payload []byte) (databasetoolssdk.CreateDatabaseToolsConnectionDetails, error) {
	concreteType, err := databaseToolsConnectionCreateConcreteType(payload)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(databasetoolssdk.CreateDatabaseToolsConnectionDetails)
	if !ok {
		return nil, fmt.Errorf("resolved CreateDatabaseToolsConnectionDetails type %s does not implement the polymorphic interface", concreteType)
	}
	return body, nil
}

func convertDatabaseToolsConnectionUpdateDetails(payload []byte) (databasetoolssdk.UpdateDatabaseToolsConnectionDetails, error) {
	concreteType, err := databaseToolsConnectionUpdateConcreteType(payload)
	if err != nil {
		return nil, err
	}

	converted := reflect.New(concreteType)
	if err := json.Unmarshal(payload, converted.Interface()); err != nil {
		return nil, fmt.Errorf("unmarshal into %s: %w", concreteType, err)
	}
	body, ok := converted.Elem().Interface().(databasetoolssdk.UpdateDatabaseToolsConnectionDetails)
	if !ok {
		return nil, fmt.Errorf("resolved UpdateDatabaseToolsConnectionDetails type %s does not implement the polymorphic interface", concreteType)
	}
	return body, nil
}

func databaseToolsConnectionCreateConcreteType(payload []byte) (reflect.Type, error) {
	connectionType, err := jsonFieldString(payload, "type")
	if err != nil {
		return nil, fmt.Errorf("decode DatabaseToolsConnection create type: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(connectionType)) {
	case "GENERIC_JDBC":
		return reflect.TypeOf(databasetoolssdk.CreateDatabaseToolsConnectionGenericJdbcDetails{}), nil
	case "POSTGRESQL":
		return reflect.TypeOf(databasetoolssdk.CreateDatabaseToolsConnectionPostgresqlDetails{}), nil
	case "MYSQL":
		return reflect.TypeOf(databasetoolssdk.CreateDatabaseToolsConnectionMySqlDetails{}), nil
	case "ORACLE_DATABASE":
		return reflect.TypeOf(databasetoolssdk.CreateDatabaseToolsConnectionOracleDatabaseDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported CreateDatabaseToolsConnectionDetails type %q", connectionType)
	}
}

func databaseToolsConnectionUpdateConcreteType(payload []byte) (reflect.Type, error) {
	connectionType, err := jsonFieldString(payload, "type")
	if err != nil {
		return nil, fmt.Errorf("decode DatabaseToolsConnection update type: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(connectionType)) {
	case "GENERIC_JDBC":
		return reflect.TypeOf(databasetoolssdk.UpdateDatabaseToolsConnectionGenericJdbcDetails{}), nil
	case "POSTGRESQL":
		return reflect.TypeOf(databasetoolssdk.UpdateDatabaseToolsConnectionPostgresqlDetails{}), nil
	case "MYSQL":
		return reflect.TypeOf(databasetoolssdk.UpdateDatabaseToolsConnectionMySqlDetails{}), nil
	case "ORACLE_DATABASE":
		return reflect.TypeOf(databasetoolssdk.UpdateDatabaseToolsConnectionOracleDatabaseDetails{}), nil
	default:
		return nil, fmt.Errorf("unsupported UpdateDatabaseToolsConnectionDetails type %q", connectionType)
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

func mutationJSONMap(value any) (map[string]any, error) {
	decoded, err := decodedJSONValueWithBoolFields(value)
	if err != nil {
		return nil, err
	}
	if decoded == nil {
		return nil, nil
	}
	decodedMap, ok := decoded.(map[string]any)
	if !ok {
		return nil, nil
	}
	return decodedMap, nil
}

func meaningfulMutationValueByPath(value any, path string) (any, bool, error) {
	values, err := mutationJSONMap(value)
	if err != nil {
		return nil, false, err
	}
	if values == nil {
		return nil, false, nil
	}
	resolved, ok := lookupMeaningfulValue(values, path)
	if !ok {
		return nil, false, nil
	}
	return resolved, true, nil
}

func decodedJSONValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal value: %w", err)
	}

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func decodedJSONValueWithBoolFields(value any) (any, error) {
	decoded, err := decodedJSONValue(value)
	if err != nil {
		return nil, err
	}
	overlayed, _ := overlayBoolFields(reflect.ValueOf(value), decoded)
	return overlayed, nil
}

func overlayBoolFields(value reflect.Value, decoded any) (any, bool) {
	value, ok := indirectValue(value)
	if !ok {
		return decoded, decoded != nil
	}
	if value.Kind() != reflect.Struct {
		return decoded, decoded != nil
	}

	decodedMap, _ := decoded.(map[string]any)
	if decodedMap == nil {
		decodedMap = map[string]any{}
	}
	hasAny := len(decodedMap) > 0

	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := typ.Field(i)
		if !fieldType.IsExported() {
			continue
		}

		fieldValue := value.Field(i)
		if fieldType.Anonymous && embeddedJSONField(fieldType) {
			embedded, embeddedHasAny := overlayBoolFields(fieldValue, decodedMap)
			if embeddedMap, ok := embedded.(map[string]any); ok {
				decodedMap = embeddedMap
				hasAny = len(decodedMap) > 0 || embeddedHasAny
			}
			continue
		}

		jsonName := fieldJSONName(fieldType)
		if jsonName == "" {
			continue
		}

		indirectField, ok := indirectValue(fieldValue)
		if !ok {
			continue
		}

		switch indirectField.Kind() {
		case reflect.Bool:
			decodedMap[jsonName] = indirectField.Bool()
			hasAny = true
		case reflect.Struct:
			childDecoded, _ := decodedMap[jsonName]
			child, childHasAny := overlayBoolFields(fieldValue, childDecoded)
			if childHasAny {
				decodedMap[jsonName] = child
				hasAny = true
			}
		}
	}

	if !hasAny {
		return nil, false
	}
	return decodedMap, true
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

	segments := strings.Split(path, ".")
	if current, ok := lookupRootScopedValue(values, segments); ok {
		return current, true
	}
	return lookupValueBySegments(values, segments)
}

func lookupRootScopedValue(values map[string]any, segments []string) (any, bool) {
	if len(segments) == 0 {
		return nil, false
	}

	switch normalizePathSegment(segments[0]) {
	case "spec":
		return lookupNamedRootValue(values, lookupSpecRootKey, segments[1:])
	case "status":
		return lookupNamedRootValue(values, lookupStatusRootKey, segments[1:])
	default:
		return nil, false
	}
}

func lookupNamedRootValue(values map[string]any, rootKey string, segments []string) (any, bool) {
	root, ok := values[rootKey].(map[string]any)
	if !ok {
		return nil, false
	}
	if len(segments) == 0 {
		return root, true
	}
	return lookupValueBySegments(root, segments)
}

func lookupValueBySegments(root map[string]any, segments []string) (any, bool) {
	current := any(root)
	for _, segment := range segments {
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
		for _, item := range concrete {
			if meaningfulValue(item) {
				return true
			}
		}
		return false
	case map[string]any:
		for _, item := range concrete {
			if meaningfulValue(item) {
				return true
			}
		}
		return false
	case bool:
		return true
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
