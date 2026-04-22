/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	databasesdk "github.com/oracle/oci-go-sdk/v65/database"
	databasetoolssdk "github.com/oracle/oci-go-sdk/v65/databasetools"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
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
	BuildCreateBody  func(context.Context, T, string) (any, error)
	BuildUpdateBody  func(context.Context, T, string, any) (any, bool, error)

	Identity        IdentityHooks[T]
	Read            ReadHooks
	TrackedRecreate TrackedRecreateHooks[T]
	StatusHooks     StatusHooks[T]
	ParityHooks     ParityHooks[T]
	Async           AsyncHooks[T]
	DeleteHooks     DeleteHooks[T]

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
