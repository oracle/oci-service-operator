package errortest

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	generator "github.com/oracle/oci-service-operator/internal/generator"
)

// APIErrorCoverageFamily records the reviewed harness family for one resource.
type APIErrorCoverageFamily string

const (
	APIErrorCoverageFamilyGeneratedRuntimePlain       APIErrorCoverageFamily = "generatedruntime-plain"
	APIErrorCoverageFamilyGeneratedRuntimeFollowUp    APIErrorCoverageFamily = "generatedruntime-follow-up"
	APIErrorCoverageFamilyGeneratedRuntimeWorkRequest APIErrorCoverageFamily = "generatedruntime-workrequest"
	APIErrorCoverageFamilyManualRuntime               APIErrorCoverageFamily = "manual-runtime"
	APIErrorCoverageFamilyLegacyAdapter               APIErrorCoverageFamily = "legacy-adapter"
)

const (
	apiErrorCoverageDefaultVersion = "v1beta1"

	deleteNotFoundGeneratedRuntime = "treat OCI 404/auth-shaped 404 and delete follow-up read misses as deleted"
	deleteNotFoundManualRuntime    = "treat OCI 404/auth-shaped 404 from direct read or delete paths as deleted"
	deleteNotFoundReadback         = "confirm deletion through helper rereads and helper-specific not-found handling"
	deleteNotFoundPendingDeletion  = "treat pending-deletion lifecycle states and 404 outcomes as deleted"

	retryableConflictGeneratedRuntime = "delete 409 conflicts reread OCI state when confirm-delete semantics are enabled"
	retryableConflictFollowUp         = "conflicts propagate through generated follow-up hooks and rereads"
	retryableConflictManualRuntime    = "conflicts rely on manual lifecycle/status requeue rather than generated follow-up helpers"
	retryableConflictReadback         = "conflicts rely on helper-specific rereads or adapter state checks"
	retryableConflictWorkRequest      = "conflicts and transient errors flow through explicit work-request resume or polling"

	manualRuntimeDelegatedDeleteDeviation = "Create/update stay in the handwritten parity runtime, but Delete delegates to generatedruntime confirm-delete semantics."
)

// APIErrorCoverageResource identifies one services.yaml-backed resource.
type APIErrorCoverageResource struct {
	Service string
	Group   string
	Version string
	Kind    string
}

func (r APIErrorCoverageResource) Key() string {
	return resourceKey(r.Service, r.Kind)
}

func (r APIErrorCoverageResource) validate() error {
	var problems []string
	if strings.TrimSpace(r.Service) == "" {
		problems = append(problems, "service is required")
	}
	if strings.TrimSpace(r.Group) == "" {
		problems = append(problems, "group is required")
	}
	if strings.TrimSpace(r.Version) == "" {
		problems = append(problems, "version is required")
	}
	if strings.TrimSpace(r.Kind) == "" {
		problems = append(problems, "kind is required")
	}
	return joinValidationProblems(problems)
}

func (r APIErrorCoverageResource) equal(other APIErrorCoverageResource) bool {
	return r.Service == other.Service &&
		r.Group == other.Group &&
		r.Version == other.Version &&
		r.Kind == other.Kind
}

// APIErrorCoverageRegistration records the reviewed coverage contract for one
// active selected resource.
type APIErrorCoverageRegistration struct {
	Resource                   APIErrorCoverageResource
	Family                     APIErrorCoverageFamily
	SupportedOperations        []Operation
	DeleteNotFoundSemantics    string
	RetryableConflictSemantics string
	Deviation                  string
}

func (r APIErrorCoverageRegistration) validate() error {
	var problems []string
	if err := r.Resource.validate(); err != nil {
		problems = append(problems, err.Error())
	}
	switch r.Family {
	case APIErrorCoverageFamilyGeneratedRuntimePlain,
		APIErrorCoverageFamilyGeneratedRuntimeFollowUp,
		APIErrorCoverageFamilyGeneratedRuntimeWorkRequest,
		APIErrorCoverageFamilyManualRuntime,
		APIErrorCoverageFamilyLegacyAdapter:
	default:
		problems = append(problems, fmt.Sprintf("family %q is not recognized", r.Family))
	}
	if err := validateSupportedOperations(r.SupportedOperations); err != nil {
		problems = append(problems, err.Error())
	}
	if strings.TrimSpace(r.DeleteNotFoundSemantics) == "" {
		problems = append(problems, "deleteNotFoundSemantics is required")
	}
	if strings.TrimSpace(r.RetryableConflictSemantics) == "" {
		problems = append(problems, "retryableConflictSemantics is required")
	}
	return joinValidationProblems(problems)
}

// APIErrorCoverageException records an explicit reviewed exemption for a kind
// that would otherwise be easy to miss from an active service.
type APIErrorCoverageException struct {
	Resource APIErrorCoverageResource
	Reason   string
}

func (e APIErrorCoverageException) validate() error {
	var problems []string
	if err := e.Resource.validate(); err != nil {
		problems = append(problems, err.Error())
	}
	if strings.TrimSpace(e.Reason) == "" {
		problems = append(problems, "reason is required")
	}
	return joinValidationProblems(problems)
}

// APIErrorCoverageInventoryItem captures the authoritative services.yaml-backed
// resource inventory that the reviewed registry must satisfy.
type APIErrorCoverageInventoryItem struct {
	Resource               APIErrorCoverageResource
	SelectionSources       []string
	ControllerStrategy     string
	ServiceManagerStrategy string
	ExceptionReason        string
}

func (i APIErrorCoverageInventoryItem) Key() string {
	return i.Resource.Key()
}

func (i APIErrorCoverageInventoryItem) RequiresRegistration() bool {
	return len(i.SelectionSources) > 0 && strings.TrimSpace(i.ExceptionReason) == ""
}

func (i APIErrorCoverageInventoryItem) RequiresException() bool {
	return strings.TrimSpace(i.ExceptionReason) != ""
}

func (i APIErrorCoverageInventoryItem) validate() error {
	var problems []string
	if err := i.Resource.validate(); err != nil {
		problems = append(problems, err.Error())
	}
	if strings.TrimSpace(i.ControllerStrategy) == "" {
		problems = append(problems, "controller strategy is required")
	}
	if strings.TrimSpace(i.ServiceManagerStrategy) == "" {
		problems = append(problems, "service-manager strategy is required")
	}
	if len(i.SelectionSources) == 0 && strings.TrimSpace(i.ExceptionReason) == "" {
		problems = append(problems, "selection sources or exception reason is required")
	}
	return joinValidationProblems(problems)
}

// APIErrorCoverageRegistry is the reviewed contract that future harness issues
// validate against the services.yaml-derived inventory.
type APIErrorCoverageRegistry struct {
	Registrations map[string]APIErrorCoverageRegistration
	Exceptions    map[string]APIErrorCoverageException
}

// ReviewedAPIErrorCoverageRegistry is the checked-in reviewed mapping from the
// active selected services.yaml surface to explicit coverage harness families or
// explicit exemptions.
var ReviewedAPIErrorCoverageRegistry = APIErrorCoverageRegistry{
	Registrations: map[string]APIErrorCoverageRegistration{
		resourceKey("containerengine", "Cluster"): reviewedRegistration(
			"containerengine",
			"containerengine",
			apiErrorCoverageDefaultVersion,
			"Cluster",
			APIErrorCoverageFamilyGeneratedRuntimeFollowUp,
			deleteNotFoundGeneratedRuntime,
			retryableConflictFollowUp,
			"Generated serviceclient uses WaitForWorkRequestWithErrorHandling for create, update, and delete follow-up.",
		),
		resourceKey("containerinstances", "ContainerInstance"): reviewedRegistration(
			"containerinstances",
			"containerinstances",
			apiErrorCoverageDefaultVersion,
			"ContainerInstance",
			APIErrorCoverageFamilyLegacyAdapter,
			deleteNotFoundManualRuntime,
			retryableConflictReadback,
			"Handwritten manager resolves tracked identity, create-or-bind lookup, and delete not-found handling through service-local helpers.",
		),
		resourceKey("core", "Drg"): reviewedRegistration(
			"core",
			"core",
			apiErrorCoverageDefaultVersion,
			"Drg",
			APIErrorCoverageFamilyLegacyAdapter,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"Create wraps the generated delegate with a post-create read fallback when OCI identity and lifecycle already project into status.",
		),
		resourceKey("core", "Instance"): reviewedRegistration(
			"core",
			"core",
			apiErrorCoverageDefaultVersion,
			"Instance",
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
		),
		resourceKey("core", "InternetGateway"): reviewedRegistration(
			"core",
			"core",
			apiErrorCoverageDefaultVersion,
			"InternetGateway",
			APIErrorCoverageFamilyManualRuntime,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			manualRuntimeDelegatedDeleteDeviation,
		),
		resourceKey("core", "NatGateway"): reviewedRegistration(
			"core",
			"core",
			apiErrorCoverageDefaultVersion,
			"NatGateway",
			APIErrorCoverageFamilyManualRuntime,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			manualRuntimeDelegatedDeleteDeviation,
		),
		resourceKey("core", "NetworkSecurityGroup"): reviewedRegistration(
			"core",
			"core",
			apiErrorCoverageDefaultVersion,
			"NetworkSecurityGroup",
			APIErrorCoverageFamilyManualRuntime,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			manualRuntimeDelegatedDeleteDeviation,
		),
		resourceKey("core", "RouteTable"): reviewedRegistration(
			"core",
			"core",
			apiErrorCoverageDefaultVersion,
			"RouteTable",
			APIErrorCoverageFamilyManualRuntime,
			deleteNotFoundManualRuntime,
			retryableConflictManualRuntime,
			"",
		),
		resourceKey("core", "SecurityList"): reviewedRegistration(
			"core",
			"core",
			apiErrorCoverageDefaultVersion,
			"SecurityList",
			APIErrorCoverageFamilyManualRuntime,
			deleteNotFoundManualRuntime,
			retryableConflictManualRuntime,
			"Explicit runtime owns nested rule normalization, stale status clearing, and SDK contract guards.",
		),
		resourceKey("core", "ServiceGateway"): reviewedRegistration(
			"core",
			"core",
			apiErrorCoverageDefaultVersion,
			"ServiceGateway",
			APIErrorCoverageFamilyManualRuntime,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			manualRuntimeDelegatedDeleteDeviation,
		),
		resourceKey("core", "Subnet"): reviewedRegistration(
			"core",
			"core",
			apiErrorCoverageDefaultVersion,
			"Subnet",
			APIErrorCoverageFamilyManualRuntime,
			deleteNotFoundManualRuntime,
			retryableConflictManualRuntime,
			"",
		),
		resourceKey("core", "Vcn"): reviewedRegistration(
			"core",
			"core",
			apiErrorCoverageDefaultVersion,
			"Vcn",
			APIErrorCoverageFamilyManualRuntime,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			manualRuntimeDelegatedDeleteDeviation,
		),
		resourceKey("database", "AutonomousDatabase"): reviewedRegistration(
			"database",
			"database",
			apiErrorCoverageDefaultVersion,
			"AutonomousDatabase",
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
		),
		resourceKey("functions", "Application"): reviewedRegistration(
			"functions",
			"functions",
			apiErrorCoverageDefaultVersion,
			"Application",
			APIErrorCoverageFamilyLegacyAdapter,
			deleteNotFoundManualRuntime,
			retryableConflictReadback,
			"Functions managers share package-local delete helpers and manual bind logic instead of generatedruntime.",
		),
		resourceKey("functions", "Function"): reviewedRegistration(
			"functions",
			"functions",
			apiErrorCoverageDefaultVersion,
			"Function",
			APIErrorCoverageFamilyLegacyAdapter,
			deleteNotFoundManualRuntime,
			retryableConflictReadback,
			"Functions managers share package-local delete helpers; Function also carries endpoint-secret side effects after OCI reconciliation.",
		),
		resourceKey("identity", "Compartment"): reviewedRegistration(
			"identity",
			"identity",
			apiErrorCoverageDefaultVersion,
			"Compartment",
			APIErrorCoverageFamilyLegacyAdapter,
			deleteNotFoundPendingDeletion,
			retryableConflictReadback,
			"Orphan-delete helper treats already deleting or deleted compartments as successful delete and short-circuits auth-shaped 404 outcomes.",
		),
		resourceKey("keymanagement", "Vault"): reviewedRegistration(
			"keymanagement",
			"keymanagement",
			apiErrorCoverageDefaultVersion,
			"Vault",
			APIErrorCoverageFamilyLegacyAdapter,
			deleteNotFoundPendingDeletion,
			"pending-deletion lifecycle and scheduled-delete handling take precedence over raw conflict classification",
			"Vault runtime wraps generatedruntime to schedule deletion windows and to treat pending deletion as terminal success.",
		),
		resourceKey("mysql", "DbSystem"): reviewedRegistration(
			"mysql",
			"mysql",
			apiErrorCoverageDefaultVersion,
			"DbSystem",
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"Credential and endpoint-secret helper layers stay outside the base generatedruntime OCI error family.",
		),
		resourceKey("nosql", "Table"): reviewedRegistration(
			"nosql",
			"nosql",
			apiErrorCoverageDefaultVersion,
			"Table",
			APIErrorCoverageFamilyLegacyAdapter,
			deleteNotFoundReadback,
			retryableConflictReadback,
			"Explicit Table runtime confirms compartment changes, updates, and deletes by reread and uses bespoke errTableNotFound handling.",
		),
		resourceKey("objectstorage", "Bucket"): reviewedRegistration(
			"objectstorage",
			"objectstorage",
			apiErrorCoverageDefaultVersion,
			"Bucket",
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"Namespace resolution may perform an extra GetNamespace OCI call before CRUD when spec.namespace and status.namespace are both empty.",
		),
		resourceKey("opensearch", "OpensearchCluster"): reviewedRegistration(
			"opensearch",
			"opensearch",
			apiErrorCoverageDefaultVersion,
			"OpensearchCluster",
			APIErrorCoverageFamilyGeneratedRuntimeFollowUp,
			deleteNotFoundGeneratedRuntime,
			retryableConflictFollowUp,
			"Generated client keeps standard read-after-write create/update follow-up and confirm-delete while sibling Opensearch kinds remain explicit strategy:none exemptions.",
		),
		resourceKey("psql", "DbSystem"): reviewedRegistration(
			"psql",
			"psql",
			apiErrorCoverageDefaultVersion,
			"DbSystem",
			APIErrorCoverageFamilyLegacyAdapter,
			deleteNotFoundReadback,
			retryableConflictReadback,
			"Active factory replaces the generated client with an explicit lifecycle and readback adapter even though generated metadata still advertises WaitForUpdatedState.",
		),
		resourceKey("queue", "Queue"): reviewedRegistration(
			"queue",
			"queue",
			apiErrorCoverageDefaultVersion,
			"Queue",
			APIErrorCoverageFamilyGeneratedRuntimeWorkRequest,
			deleteNotFoundReadback,
			retryableConflictWorkRequest,
			"Queue runtime persists create, update, and delete work-request OCIDs and recovers Queue identity from work-request payloads.",
		),
		resourceKey("redis", "RedisCluster"): reviewedRegistration(
			"redis",
			"redis",
			apiErrorCoverageDefaultVersion,
			"RedisCluster",
			APIErrorCoverageFamilyLegacyAdapter,
			deleteNotFoundReadback,
			"delete 409 conflicts reread live RedisCluster state before deciding retry or success",
			"Create/update keep the generated read-after-write baseline, while delete wraps the generated client with a live-state delete guard before and after conflicts.",
		),
		resourceKey("streaming", "Stream"): reviewedRegistration(
			"streaming",
			"streaming",
			apiErrorCoverageDefaultVersion,
			"Stream",
			APIErrorCoverageFamilyGeneratedRuntimeFollowUp,
			deleteNotFoundGeneratedRuntime,
			retryableConflictFollowUp,
			"Update follow-up uses WaitForUpdatedState and endpoint-secret reconciliation is a side effect outside the base CRUD harness.",
		),
	},
	Exceptions: map[string]APIErrorCoverageException{
		resourceKey("keymanagement", "Key"): reviewedException(
			"keymanagement",
			"keymanagement",
			apiErrorCoverageDefaultVersion,
			"Key",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("keymanagement", "KeyVersion"): reviewedException(
			"keymanagement",
			"keymanagement",
			apiErrorCoverageDefaultVersion,
			"KeyVersion",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("keymanagement", "ReplicationStatus"): reviewedException(
			"keymanagement",
			"keymanagement",
			apiErrorCoverageDefaultVersion,
			"ReplicationStatus",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("keymanagement", "WrappingKey"): reviewedException(
			"keymanagement",
			"keymanagement",
			apiErrorCoverageDefaultVersion,
			"WrappingKey",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("opensearch", "Manifest"): reviewedException(
			"opensearch",
			"opensearch",
			apiErrorCoverageDefaultVersion,
			"Manifest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("opensearch", "OpensearchClusterBackup"): reviewedException(
			"opensearch",
			"opensearch",
			apiErrorCoverageDefaultVersion,
			"OpensearchClusterBackup",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("opensearch", "OpensearchOpensearchVersion"): reviewedException(
			"opensearch",
			"opensearch",
			apiErrorCoverageDefaultVersion,
			"OpensearchOpensearchVersion",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("opensearch", "WorkRequest"): reviewedException(
			"opensearch",
			"opensearch",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("opensearch", "WorkRequestError"): reviewedException(
			"opensearch",
			"opensearch",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("opensearch", "WorkRequestLog"): reviewedException(
			"opensearch",
			"opensearch",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
	},
}

// LoadCheckedInAPIErrorCoverageInventory resolves the repo-local services.yaml
// and derives the authoritative active inventory for API-error coverage.
func LoadCheckedInAPIErrorCoverageInventory() ([]APIErrorCoverageInventoryItem, error) {
	servicesPath, err := checkedInServicesConfigPath()
	if err != nil {
		return nil, err
	}
	return LoadAPIErrorCoverageInventory(servicesPath)
}

// LoadAPIErrorCoverageInventory loads a services config and derives the active
// API-error coverage inventory from it.
func LoadAPIErrorCoverageInventory(path string) ([]APIErrorCoverageInventoryItem, error) {
	cfg, err := generator.LoadConfig(path)
	if err != nil {
		return nil, err
	}
	return BuildAPIErrorCoverageInventory(cfg)
}

// BuildAPIErrorCoverageInventory derives the authoritative API-error coverage
// inventory from default-active controller-backed services.
func BuildAPIErrorCoverageInventory(cfg *generator.Config) ([]APIErrorCoverageInventoryItem, error) {
	if cfg == nil {
		return nil, fmt.Errorf("generator config is nil")
	}

	itemsByKey := make(map[string]*APIErrorCoverageInventoryItem)
	for _, service := range cfg.DefaultActiveServices() {
		if !service.IsControllerBacked() {
			continue
		}
		if mode := service.DefaultSelectionMode(); mode != generator.SelectionModeExplicit {
			return nil, fmt.Errorf(
				"active controller-backed service %q uses selection.mode %q; API-error coverage inventory requires %q",
				service.Service,
				mode,
				generator.SelectionModeExplicit,
			)
		}

		version := service.VersionOrDefault(cfg.DefaultVersion)
		addSelection := func(kind string, source string) {
			item := ensureInventoryItem(itemsByKey, service, version, kind)
			item.SelectionSources = appendUniqueString(item.SelectionSources, source)
			item.ControllerStrategy = service.ControllerGenerationStrategyFor(kind)
			item.ServiceManagerStrategy = service.ServiceManagerGenerationStrategyFor(kind)
		}

		for _, kind := range service.DefaultIncludeKinds() {
			addSelection(kind, "selection.includeKinds")
		}
		for _, split := range service.PackageSplits {
			for _, kind := range split.IncludeKinds {
				addSelection(strings.TrimSpace(kind), fmt.Sprintf("packageSplits[%s].includeKinds", strings.TrimSpace(split.Name)))
			}
		}

		for _, override := range service.Generation.Resources {
			kind := strings.TrimSpace(override.Kind)
			if kind == "" {
				continue
			}
			controllerStrategy := service.ControllerGenerationStrategyFor(kind)
			serviceManagerStrategy := service.ServiceManagerGenerationStrategyFor(kind)
			if controllerStrategy != generator.GenerationStrategyNone && serviceManagerStrategy != generator.GenerationStrategyNone {
				continue
			}

			item := ensureInventoryItem(itemsByKey, service, version, kind)
			item.ControllerStrategy = controllerStrategy
			item.ServiceManagerStrategy = serviceManagerStrategy
			item.ExceptionReason = fmt.Sprintf(
				"effective controller.strategy=%q and serviceManager.strategy=%q keep this kind outside the active controller-backed API-error surface",
				controllerStrategy,
				serviceManagerStrategy,
			)
		}
	}

	items := make([]APIErrorCoverageInventoryItem, 0, len(itemsByKey))
	for _, item := range itemsByKey {
		item.SelectionSources = sortedStrings(item.SelectionSources)
		items = append(items, *item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Key() < items[j].Key()
	})
	return items, nil
}

// ValidateCheckedInAPIErrorCoverageRegistry validates the reviewed registry
// against the checked-in services.yaml inventory.
func ValidateCheckedInAPIErrorCoverageRegistry() error {
	inventory, err := LoadCheckedInAPIErrorCoverageInventory()
	if err != nil {
		return err
	}
	return ReviewedAPIErrorCoverageRegistry.ValidateInventory(inventory)
}

// ValidateInventory ensures the reviewed registry covers the current active
// inventory without silent gaps or stale extras.
func (r APIErrorCoverageRegistry) ValidateInventory(inventory []APIErrorCoverageInventoryItem) error {
	var problems []string

	seenRegistrations := make(map[string]APIErrorCoverageRegistration, len(r.Registrations))
	for key, registration := range r.Registrations {
		if err := registration.validate(); err != nil {
			problems = append(problems, fmt.Sprintf("registration %s: %v", key, err))
		}
		if key != registration.Resource.Key() {
			problems = append(problems, fmt.Sprintf("registration key %q does not match resource key %q", key, registration.Resource.Key()))
		}
		seenRegistrations[key] = registration
	}

	seenExceptions := make(map[string]APIErrorCoverageException, len(r.Exceptions))
	for key, exception := range r.Exceptions {
		if err := exception.validate(); err != nil {
			problems = append(problems, fmt.Sprintf("exception %s: %v", key, err))
		}
		if key != exception.Resource.Key() {
			problems = append(problems, fmt.Sprintf("exception key %q does not match resource key %q", key, exception.Resource.Key()))
		}
		seenExceptions[key] = exception
	}

	for key := range seenRegistrations {
		if _, ok := seenExceptions[key]; ok {
			problems = append(problems, fmt.Sprintf("%s appears in both registrations and exceptions", key))
		}
	}

	expectedRegistrations := make(map[string]APIErrorCoverageInventoryItem)
	expectedExceptions := make(map[string]APIErrorCoverageInventoryItem)
	for _, item := range inventory {
		if err := item.validate(); err != nil {
			problems = append(problems, fmt.Sprintf("inventory %s: %v", item.Key(), err))
			continue
		}
		switch {
		case item.RequiresException():
			expectedExceptions[item.Key()] = item
		case item.RequiresRegistration():
			expectedRegistrations[item.Key()] = item
		}
	}

	for key, item := range expectedRegistrations {
		registration, ok := seenRegistrations[key]
		if !ok {
			problems = append(problems, fmt.Sprintf("missing reviewed registration for %s", key))
			continue
		}
		if !registration.Resource.equal(item.Resource) {
			problems = append(problems, fmt.Sprintf("registration %s resource identity = %#v, want %#v", key, registration.Resource, item.Resource))
		}
	}

	for key, item := range expectedExceptions {
		exception, ok := seenExceptions[key]
		if !ok {
			problems = append(problems, fmt.Sprintf("missing reviewed exception for %s", key))
			continue
		}
		if !exception.Resource.equal(item.Resource) {
			problems = append(problems, fmt.Sprintf("exception %s resource identity = %#v, want %#v", key, exception.Resource, item.Resource))
		}
	}

	for key := range seenRegistrations {
		if _, ok := expectedRegistrations[key]; !ok {
			problems = append(problems, fmt.Sprintf("registration %s is not part of the current active services.yaml inventory", key))
		}
	}
	for key := range seenExceptions {
		if _, ok := expectedExceptions[key]; !ok {
			problems = append(problems, fmt.Sprintf("exception %s is not part of the current active services.yaml inventory", key))
		}
	}

	sort.Strings(problems)
	if len(problems) > 0 {
		return fmt.Errorf("API-error coverage registry validation failed:\n- %s", strings.Join(problems, "\n- "))
	}
	return nil
}

func checkedInServicesConfigPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve checked-in services config path: runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "internal", "generator", "config", "services.yaml")), nil
}

func reviewedRegistration(
	service string,
	group string,
	version string,
	kind string,
	family APIErrorCoverageFamily,
	deleteNotFoundSemantics string,
	retryableConflictSemantics string,
	deviation string,
) APIErrorCoverageRegistration {
	return APIErrorCoverageRegistration{
		Resource: APIErrorCoverageResource{
			Service: service,
			Group:   group,
			Version: version,
			Kind:    kind,
		},
		Family:                     family,
		SupportedOperations:        crudOperations(),
		DeleteNotFoundSemantics:    deleteNotFoundSemantics,
		RetryableConflictSemantics: retryableConflictSemantics,
		Deviation:                  deviation,
	}
}

func reviewedException(
	service string,
	group string,
	version string,
	kind string,
	reason string,
) APIErrorCoverageException {
	return APIErrorCoverageException{
		Resource: APIErrorCoverageResource{
			Service: service,
			Group:   group,
			Version: version,
			Kind:    kind,
		},
		Reason: reason,
	}
}

func ensureInventoryItem(
	items map[string]*APIErrorCoverageInventoryItem,
	service generator.ServiceConfig,
	version string,
	kind string,
) *APIErrorCoverageInventoryItem {
	key := resourceKey(service.Service, kind)
	if item, ok := items[key]; ok {
		return item
	}

	item := &APIErrorCoverageInventoryItem{
		Resource: APIErrorCoverageResource{
			Service: service.Service,
			Group:   service.Group,
			Version: version,
			Kind:    kind,
		},
	}
	items[key] = item
	return item
}

func crudOperations() []Operation {
	return []Operation{
		OperationRead,
		OperationCreate,
		OperationUpdate,
		OperationDelete,
	}
}

func validateSupportedOperations(operations []Operation) error {
	if len(operations) == 0 {
		return fmt.Errorf("supportedOperations must not be empty")
	}

	seen := make(map[Operation]struct{}, len(operations))
	for _, operation := range operations {
		switch operation {
		case OperationRead, OperationCreate, OperationUpdate, OperationDelete:
		default:
			return fmt.Errorf("supported operation %q is not recognized", operation)
		}
		if _, ok := seen[operation]; ok {
			return fmt.Errorf("supported operation %q is listed more than once", operation)
		}
		seen[operation] = struct{}{}
	}
	return nil
}

func resourceKey(service string, kind string) string {
	return fmt.Sprintf("%s/%s", strings.TrimSpace(service), strings.TrimSpace(kind))
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func sortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func joinValidationProblems(problems []string) error {
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("%s", strings.Join(problems, "; "))
}
