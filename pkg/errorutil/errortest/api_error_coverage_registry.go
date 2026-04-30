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
	Registrations: mergeReviewedRegistrationSets(
		map[string]APIErrorCoverageRegistration{
			resourceKey("accessgovernancecp", "GovernanceInstance"): reviewedRegistration(
				"accessgovernancecp",
				"accessgovernancecp",
				apiErrorCoverageDefaultVersion,
				"GovernanceInstance",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"GovernanceInstance uses plain generatedruntime lifecycle rereads with a handwritten hook layer for tenant-safe pre-create reuse, explicit mutable-field shaping, and create-only idcsAccessToken normalization; the SDK exposes work-request headers but no service-local work-request reader.",
			),
			resourceKey("adm", "KnowledgeBase"): reviewedRegistration(
				"adm",
				"adm",
				apiErrorCoverageDefaultVersion,
				"KnowledgeBase",
				APIErrorCoverageFamilyGeneratedRuntimeWorkRequest,
				deleteNotFoundReadback,
				retryableConflictWorkRequest,
				"KnowledgeBase runtime persists create, update, and delete work-request IDs, recovers KnowledgeBase identity from work-request resources, and bounds pre-create reuse to exact compartmentId plus displayName matches.",
			),
			resourceKey("aidocument", "Project"): reviewedRegistration(
				"aidocument",
				"aidocument",
				apiErrorCoverageDefaultVersion,
				"Project",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Initial Project rollout keeps plain generatedruntime CRUD and lifecycle rereads even though the SDK also exposes work-request identifiers and a separate ChangeProjectCompartment action.",
			),
			resourceKey("ailanguage", "Project"): reviewedRegistration(
				"ailanguage",
				"ailanguage",
				apiErrorCoverageDefaultVersion,
				"Project",
				APIErrorCoverageFamilyGeneratedRuntimeWorkRequest,
				deleteNotFoundReadback,
				retryableConflictWorkRequest,
				"Handwritten Project create/update/delete now persist work-request OCIDs in shared async status, poll GetWorkRequest, and keep delete confirmation through GetProject plus ListProjects fallback.",
			),
			resourceKey("aispeech", "TranscriptionJob"): reviewedRegistration(
				"aispeech",
				"aispeech",
				apiErrorCoverageDefaultVersion,
				"TranscriptionJob",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"TranscriptionJob keeps generatedruntime request projection and rereads, but a small handwritten layer now owns FAILED/CANCELING/CANCELED lifecycle mapping and delete confirmation until GetTranscriptionJob returns 404/auth-shaped 404.",
			),
			resourceKey("aivision", "Project"): reviewedRegistration(
				"aivision",
				"aivision",
				apiErrorCoverageDefaultVersion,
				"Project",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Initial Project rollout keeps plain generatedruntime CRUD and lifecycle rereads even though the SDK also exposes work-request identifiers and a separate ChangeProjectCompartment action.",
			),
			resourceKey("analytics", "AnalyticsInstance"): reviewedRegistration(
				"analytics",
				"analytics",
				apiErrorCoverageDefaultVersion,
				"AnalyticsInstance",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Handwritten update-body shaping narrows AnalyticsInstance updates to UpdateAnalyticsInstanceDetails fields while create, read, and delete OCI errors still flow through shared generatedruntime CRUD and confirm-delete handling.",
			),
			resourceKey("apiaccesscontrol", "PrivilegedApiControl"): reviewedRegistration(
				"apiaccesscontrol",
				"apiaccesscontrol",
				apiErrorCoverageDefaultVersion,
				"PrivilegedApiControl",
				APIErrorCoverageFamilyGeneratedRuntimeWorkRequest,
				deleteNotFoundReadback,
				retryableConflictWorkRequest,
				"PrivilegedApiControl runtime persists create, update, and delete work-request IDs in shared async status, bounds pre-create reuse to exact compartmentId plus displayName plus resourceType matches, preserves clear-to-empty update semantics, and narrows delete request projection so spec.description is not sent as the delete reason query field.",
			),
			resourceKey("apiplatform", "ApiPlatformInstance"): reviewedRegistration(
				"apiplatform",
				"apiplatform",
				apiErrorCoverageDefaultVersion,
				"ApiPlatformInstance",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"The published ApiPlatformInstance runtime keeps generatedruntime CRUD, lifecycle rereads, and confirm-delete behavior, while a small handwritten hook narrows list lookup semantics and clears ChangeApiPlatformInstanceCompartment from the reviewed surface.",
			),
			resourceKey("apmconfig", "Config"): reviewedRegistration(
				"apmconfig",
				"apmconfig",
				apiErrorCoverageDefaultVersion,
				"Config",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Config keeps generatedruntime CRUD and confirm-delete handling, while a handwritten polymorphic runtime hook layer dispatches subtype-specific request bodies, mirrors apmDomainId into status, and rejects incompatible top-level fields before OCI calls.",
			),
			resourceKey("apmcontrolplane", "ApmDomain"): reviewedRegistration(
				"apmcontrolplane",
				"apmcontrolplane",
				apiErrorCoverageDefaultVersion,
				"ApmDomain",
				APIErrorCoverageFamilyGeneratedRuntimeWorkRequest,
				deleteNotFoundReadback,
				retryableConflictWorkRequest,
				"ApmDomain runtime persists create, update, and delete work-request IDs in shared async status, recovers ApmDomain identity from work-request resources, and bounds pre-create reuse to exact compartmentId plus displayName matches while keeping ChangeApmDomainCompartment out of the published surface.",
			),
			resourceKey("apmtraces", "ScheduledQuery"): reviewedRegistration(
				"apmtraces",
				"apmtraces",
				apiErrorCoverageDefaultVersion,
				"ScheduledQuery",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"ScheduledQuery keeps lifecycle-based generatedruntime CRUD and confirm-delete handling, while a handwritten hook layer rewires the SDK displayName filter to spec.scheduledQueryName, bounds pre-create reuse to exact-name matches inside one apmDomainId, and mirrors apmDomainId into status because OCI does not echo it back.",
			),
			resourceKey("apmsynthetics", "Script"): reviewedRegistration(
				"apmsynthetics",
				"apmsynthetics",
				apiErrorCoverageDefaultVersion,
				"Script",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Script keeps generatedruntime CRUD and confirm-delete handling, while a handwritten hook layer scopes requests by apmDomainId, mirrors that bound domain into status, settles synchronous create and update rereads as Active, and bounds pre-create reuse to exact apmDomainId plus displayName plus contentType matches.",
			),
			resourceKey("bds", "BdsInstance"): reviewedRegistration(
				"bds",
				"bds",
				apiErrorCoverageDefaultVersion,
				"BdsInstance",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Initial BdsInstance rollout keeps plain generatedruntime CRUD and lifecycle rereads even though the SDK also exposes work-request identifiers and a separate ChangeBdsInstanceCompartment action.",
			),
			resourceKey("budget", "Budget"): reviewedRegistration(
				"budget",
				"budget",
				apiErrorCoverageDefaultVersion,
				"Budget",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Initial Budget rollout keeps plain generatedruntime CRUD, rereads, and confirm-delete handling with async.strategy=none because the service exposes direct CRUD without service-local work requests.",
			),
			resourceKey("capacitymanagement", "OccCapacityRequest"): reviewedRegistration(
				"capacitymanagement",
				"capacitymanagement",
				apiErrorCoverageDefaultVersion,
				"OccCapacityRequest",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"OccCapacityRequest keeps lifecycle-based generatedruntime CRUD and confirm-delete handling, disables untracked list adoption entirely, and preserves OCI retry-after hints across create, update, and delete requeues.",
			),
			resourceKey("clusterplacementgroups", "ClusterPlacementGroup"): reviewedRegistration(
				"clusterplacementgroups",
				"clusterplacementgroups",
				apiErrorCoverageDefaultVersion,
				"ClusterPlacementGroup",
				APIErrorCoverageFamilyGeneratedRuntimeWorkRequest,
				deleteNotFoundReadback,
				retryableConflictWorkRequest,
				"ClusterPlacementGroup runtime persists CRUD work-request IDs in shared async status, bounds pre-create reuse to exact compartmentId plus name, availabilityDomain, and clusterPlacementGroupType matches, and keeps activate/deactivate/compartment-change helpers out of the published surface.",
			),
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
			resourceKey("containerengine", "NodePool"): reviewedRegistration(
				"containerengine",
				"containerengine",
				apiErrorCoverageDefaultVersion,
				"NodePool",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Handwritten request-body shaping still delegates create, update, and delete OCI errors to the shared generatedruntime CRUD and confirm-delete paths.",
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
			resourceKey("databasetools", "DatabaseToolsConnection"): reviewedRegistration(
				"databasetools",
				"databasetools",
				apiErrorCoverageDefaultVersion,
				"DatabaseToolsConnection",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Initial DatabaseToolsConnection rollout keeps plain generatedruntime CRUD and lifecycle rereads even though the service also exposes endpoint services, private endpoints, work-request identifiers, and connection validation helpers.",
			),
			resourceKey("databasemigration", "Connection"): reviewedRegistration(
				"databasemigration",
				"databasemigration",
				apiErrorCoverageDefaultVersion,
				"Connection",
				APIErrorCoverageFamilyGeneratedRuntimeWorkRequest,
				deleteNotFoundReadback,
				retryableConflictWorkRequest,
				"Connection runtime persists create, update, and delete work-request OCIDs in shared async status, recovers Connection identity from work-request resources, and bounds pre-create reuse to exact compartmentId plus displayName matches narrowed by connectionType, technologyType, and observable type-specific identity fields.",
			),
			resourceKey("datascience", "Project"): reviewedRegistration(
				"datascience",
				"datascience",
				apiErrorCoverageDefaultVersion,
				"Project",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Initial Project rollout keeps plain generatedruntime CRUD and lifecycle rereads even though the SDK also exposes work-request identifiers and a separate ChangeProjectCompartment action.",
			),
			resourceKey("dashboardservice", "DashboardGroup"): reviewedRegistration(
				"dashboardservice",
				"dashboardservice",
				apiErrorCoverageDefaultVersion,
				"DashboardGroup",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"DashboardGroup keeps generatedruntime CRUD and lifecycle rereads, while a small handwritten seam skips unsafe pre-create reuse when displayName is empty, preserves explicit empty-map clears for tag maps, and treats empty-string displayName/description values as omission until the spec can distinguish clear from omit.",
			),
			resourceKey("dataflow", "Application"): reviewedRegistration(
				"dataflow",
				"dataflow",
				apiErrorCoverageDefaultVersion,
				"Application",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Application now keeps delete rereads, OCI error normalization, and request-ID projection inside bounded generatedruntime delete hooks; only a small wrapper still clears tracked identity before recreate when a reread reports DELETED.",
			),
			resourceKey("datalabelingservice", "Dataset"): reviewedRegistration(
				"datalabelingservice",
				"datalabelingservice",
				apiErrorCoverageDefaultVersion,
				"Dataset",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Dataset keeps generatedruntime lifecycle rereads and confirm-delete handling, while a small handwritten layer shapes polymorphic create bodies, narrows list reuse to exact compartmentId plus annotationFormat plus displayName matches, and preserves clear-to-empty update intent for UpdateDatasetDetails.",
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
				APIErrorCoverageFamilyGeneratedRuntimeWorkRequest,
				deleteNotFoundReadback,
				"create/update conflicts and transient errors flow through explicit Redis work-request resume or polling, while delete 409 conflicts reread live RedisCluster state before deciding retry or success",
				"Handwritten create/update/delete now own Redis work-request polling while delete still wraps the runtime with a live-state delete guard before and after conflicts.",
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
		map[string]APIErrorCoverageRegistration{
			resourceKey("generativeai", "DedicatedAiCluster"): reviewedRegistration(
				"generativeai",
				"generativeai",
				apiErrorCoverageDefaultVersion,
				"DedicatedAiCluster",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Service-local create wrapping skips displayName-based reuse when spec.displayName is empty, clears stale tracked identifiers after pre-create rereads return not-found or DELETED, and then re-enters shared generatedruntime CRUD plus confirm-delete handling.",
			),
			resourceKey("generativeai", "Model"): reviewedRegistration(
				"generativeai",
				"generativeai",
				apiErrorCoverageDefaultVersion,
				"Model",
				APIErrorCoverageFamilyGeneratedRuntimePlain,
				deleteNotFoundGeneratedRuntime,
				retryableConflictGeneratedRuntime,
				"Service-local create wrapping skips displayName-based reuse when spec.displayName is empty, clears stale tracked identifiers after pre-create rereads return not-found or DELETED, and otherwise re-enters shared generatedruntime CRUD plus confirm-delete handling with formal baseModelId and fineTuneDetails identity matching.",
			),
		},
		reviewedRegistrationSet(
			"artifacts",
			"artifacts",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"ContainerImageSignature",
			"ContainerRepository",
			"Repository",
		),
		reviewedRegistrationSet(
			"bastion",
			"bastion",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Bastion",
			"Session",
		),
		reviewedRegistrationSet(
			"certificatesmanagement",
			"certificatesmanagement",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"CaBundle",
		),
		reviewedRegistrationSet(
			"devops",
			"devops",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"BuildPipeline",
			"DeployArtifact",
			"DeployPipeline",
			"Project",
			"Repository",
			"Trigger",
		),
		reviewedRegistrationSet(
			"dns",
			"dns",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"SteeringPolicy",
			"SteeringPolicyAttachment",
			"TsigKey",
			"View",
			"Zone",
		),
		reviewedRegistrationSet(
			"events",
			"events",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Rule",
		),
		reviewedRegistrationSet(
			"healthchecks",
			"healthchecks",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"HttpMonitor",
			"PingMonitor",
		),
		reviewedRegistrationSet(
			"integration",
			"integration",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"IntegrationInstance",
		),
		reviewedRegistrationSet(
			"keymanagement",
			"keymanagement",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"EkmsPrivateEndpoint",
		),
		reviewedRegistrationSet(
			"limits",
			"limits",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Quota",
		),
		reviewedRegistrationSet(
			"managedkafka",
			"managedkafka",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"KafkaCluster",
			"KafkaClusterConfig",
		),
		reviewedRegistrationSet(
			"networkloadbalancer",
			"networkloadbalancer",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"Initial Network Load Balancer rollout uses generatedruntime CRUD; name-scoped child resources may still need resource-local identity hooks before live validation.",
			"Backend",
			"BackendSet",
			"Listener",
			"NetworkLoadBalancer",
		),
		reviewedRegistrationSet(
			"ons",
			"ons",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Subscription",
			"Topic",
		),
		reviewedRegistrationSet(
			"sch",
			"sch",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"ServiceConnector",
		),
		reviewedRegistrationSet(
			"servicecatalog",
			"servicecatalog",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"PrivateApplication",
			"ServiceCatalog",
		),
		reviewedRegistrationSet(
			"email",
			"email",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Dkim",
			"EmailDomain",
			"Sender",
			"Suppression",
		),
		reviewedRegistrationSet(
			"generativeai",
			"generativeai",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Endpoint",
		),
		reviewedRegistrationSet(
			"loadbalancer",
			"loadbalancer",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Backend",
			"BackendSet",
			"Certificate",
			"Hostname",
			"Listener",
			"LoadBalancer",
			"PathRouteSet",
			"RoutingPolicy",
			"RuleSet",
			"SSLCipherSuite",
		),
		reviewedRegistrationSet(
			"logging",
			"logging",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Log",
			"LogGroup",
			"LogSavedSearch",
			"UnifiedAgentConfiguration",
		),
		reviewedRegistrationSet(
			"marketplace",
			"marketplace",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"AcceptedAgreement",
			"Publication",
		),
		reviewedRegistrationSet(
			"monitoring",
			"monitoring",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Alarm",
			"AlarmSuppression",
		),
		reviewedRegistrationSet(
			"ocvp",
			"ocvp",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Cluster",
			"EsxiHost",
			"Sddc",
		),
		reviewedRegistrationSet(
			"oda",
			"oda",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"AuthenticationProvider",
			"Channel",
			"DigitalAssistant",
			"ImportedPackage",
			"OdaInstance",
			"OdaInstanceAttachment",
			"OdaPrivateEndpoint",
			"OdaPrivateEndpointAttachment",
			"OdaPrivateEndpointScanProxy",
			"Skill",
			"SkillParameter",
			"Translator",
		),
		reviewedRegistrationSet(
			"usageapi",
			"usageapi",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"CustomTable",
			"Query",
			"Schedule",
			"UsageCarbonEmissionsQuery",
		),
		reviewedRegistrationSet(
			"governancerulescontrolplane",
			"governancerulescontrolplane",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"GovernanceRule",
			"InclusionCriterion",
		),
		reviewedRegistrationSet(
			"iot",
			"iot",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"DigitalTwinAdapter",
			"DigitalTwinInstance",
			"DigitalTwinModel",
			"DigitalTwinRelationship",
			"IotDomain",
			"IotDomainGroup",
		),
		reviewedRegistrationSet(
			"licensemanager",
			"licensemanager",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"LicenseRecord",
			"ProductLicense",
		),
		reviewedRegistrationSet(
			"limitsincrease",
			"limitsincrease",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"LimitsIncreaseRequest",
		),
		reviewedRegistrationSet(
			"lockbox",
			"lockbox",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"ApprovalTemplate",
			"Lockbox",
		),
		reviewedRegistrationSet(
			"loganalytics",
			"loganalytics",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"IngestTimeRule",
			"LogAnalyticsEmBridge",
			"LogAnalyticsEntity",
			"LogAnalyticsEntityType",
			"LogAnalyticsLogGroup",
			"LogAnalyticsObjectCollectionRule",
			"ScheduledTask",
		),
		reviewedRegistrationSet(
			"managementagent",
			"managementagent",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"DataSource",
			"ManagementAgentInstallKey",
			"NamedCredential",
		),
		reviewedRegistrationSet(
			"managementdashboard",
			"managementdashboard",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"ManagementDashboard",
			"ManagementSavedSearch",
		),
		reviewedRegistrationSet(
			"marketplaceprivateoffer",
			"marketplaceprivateoffer",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Attachment",
			"Offer",
		),
		reviewedRegistrationSet(
			"marketplacepublisher",
			"marketplacepublisher",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Artifact",
			"Listing",
			"ListingRevision",
			"ListingRevisionAttachment",
			"ListingRevisionNote",
			"ListingRevisionPackage",
			"Term",
			"TermVersion",
		),
		reviewedRegistrationSet(
			"operatoraccesscontrol",
			"operatoraccesscontrol",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"OperatorControl",
			"OperatorControlAssignment",
		),
		reviewedRegistrationSet(
			"opsi",
			"opsi",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"AwrHub",
			"AwrHubSource",
			"ChargebackPlan",
			"ChargebackPlanReport",
			"DatabaseInsight",
			"EnterpriseManagerBridge",
			"ExadataInsight",
			"HostInsight",
			"NewsReport",
			"OperationsInsightsPrivateEndpoint",
			"OperationsInsightsWarehouse",
			"OperationsInsightsWarehouseUser",
			"OpsiConfiguration",
		),
		reviewedRegistrationSet(
			"optimizer",
			"optimizer",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Profile",
		),
		reviewedRegistrationSet(
			"osmanagementhub",
			"osmanagementhub",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"LifecycleEnvironment",
			"ManagedInstanceGroup",
			"ManagementStation",
			"Profile",
			"ScheduledJob",
			"SoftwareSource",
		),
		reviewedRegistrationSet(
			"recovery",
			"recovery",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"ProtectedDatabase",
			"ProtectionPolicy",
			"RecoveryServiceSubnet",
		),
		reviewedRegistrationSet(
			"resourceanalytics",
			"resourceanalytics",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"MonitoredRegion",
			"ResourceAnalyticsInstance",
			"TenancyAttachment",
		),
		reviewedRegistrationSet(
			"resourcescheduler",
			"resourcescheduler",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"Schedule",
		),
		reviewedRegistrationSet(
			"securityattribute",
			"securityattribute",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"SecurityAttribute",
			"SecurityAttributeNamespace",
		),
		reviewedRegistrationSet(
			"stackmonitoring",
			"stackmonitoring",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"AlarmCondition",
			"BaselineableMetric",
			"Config",
			"DiscoveryJob",
			"MaintenanceWindow",
			"MetricExtension",
			"MonitoredResource",
			"MonitoredResourceType",
			"MonitoringTemplate",
			"ProcessSet",
		),
		reviewedRegistrationSet(
			"waa",
			"waa",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"WebAppAcceleration",
			"WebAppAccelerationPolicy",
		),
		reviewedRegistrationSet(
			"waas",
			"waas",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"AddressList",
			"Certificate",
			"CustomProtectionRule",
			"HttpRedirect",
			"WaasPolicy",
		),
		reviewedRegistrationSet(
			"waf",
			"waf",
			apiErrorCoverageDefaultVersion,
			APIErrorCoverageFamilyGeneratedRuntimePlain,
			deleteNotFoundGeneratedRuntime,
			retryableConflictGeneratedRuntime,
			"",
			"NetworkAddressList",
			"WebAppFirewall",
			"WebAppFirewallPolicy",
		),
	),
	Exceptions: map[string]APIErrorCoverageException{
		resourceKey("accessgovernancecp", "GovernanceInstanceConfiguration"): reviewedException(
			"accessgovernancecp",
			"accessgovernancecp",
			apiErrorCoverageDefaultVersion,
			"GovernanceInstanceConfiguration",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("accessgovernancecp", "SenderConfig"): reviewedException(
			"accessgovernancecp",
			"accessgovernancecp",
			apiErrorCoverageDefaultVersion,
			"SenderConfig",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("adm", "ApplicationDependencyRecommendation"): reviewedException(
			"adm",
			"adm",
			apiErrorCoverageDefaultVersion,
			"ApplicationDependencyRecommendation",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("adm", "ApplicationDependencyVulnerability"): reviewedException(
			"adm",
			"adm",
			apiErrorCoverageDefaultVersion,
			"ApplicationDependencyVulnerability",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("adm", "RemediationRecipe"): reviewedException(
			"adm",
			"adm",
			apiErrorCoverageDefaultVersion,
			"RemediationRecipe",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("adm", "RemediationRun"): reviewedException(
			"adm",
			"adm",
			apiErrorCoverageDefaultVersion,
			"RemediationRun",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("adm", "Stage"): reviewedException(
			"adm",
			"adm",
			apiErrorCoverageDefaultVersion,
			"Stage",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("adm", "Vulnerability"): reviewedException(
			"adm",
			"adm",
			apiErrorCoverageDefaultVersion,
			"Vulnerability",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("adm", "VulnerabilityAudit"): reviewedException(
			"adm",
			"adm",
			apiErrorCoverageDefaultVersion,
			"VulnerabilityAudit",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("adm", "WorkRequest"): reviewedException(
			"adm",
			"adm",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("adm", "WorkRequestError"): reviewedException(
			"adm",
			"adm",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("adm", "WorkRequestLog"): reviewedException(
			"adm",
			"adm",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aidocument", "Model"): reviewedException(
			"aidocument",
			"aidocument",
			apiErrorCoverageDefaultVersion,
			"Model",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aidocument", "ProcessorJob"): reviewedException(
			"aidocument",
			"aidocument",
			apiErrorCoverageDefaultVersion,
			"ProcessorJob",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aidocument", "WorkRequest"): reviewedException(
			"aidocument",
			"aidocument",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aidocument", "WorkRequestError"): reviewedException(
			"aidocument",
			"aidocument",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aidocument", "WorkRequestLog"): reviewedException(
			"aidocument",
			"aidocument",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apiaccesscontrol", "ApiMetadata"): reviewedException(
			"apiaccesscontrol",
			"apiaccesscontrol",
			apiErrorCoverageDefaultVersion,
			"ApiMetadata",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apiaccesscontrol", "ApiMetadataByEntityType"): reviewedException(
			"apiaccesscontrol",
			"apiaccesscontrol",
			apiErrorCoverageDefaultVersion,
			"ApiMetadataByEntityType",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apiaccesscontrol", "PrivilegedApiRequest"): reviewedException(
			"apiaccesscontrol",
			"apiaccesscontrol",
			apiErrorCoverageDefaultVersion,
			"PrivilegedApiRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apiaccesscontrol", "WorkRequest"): reviewedException(
			"apiaccesscontrol",
			"apiaccesscontrol",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apiaccesscontrol", "WorkRequestError"): reviewedException(
			"apiaccesscontrol",
			"apiaccesscontrol",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apiaccesscontrol", "WorkRequestLog"): reviewedException(
			"apiaccesscontrol",
			"apiaccesscontrol",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("ailanguage", "Endpoint"): reviewedException(
			"ailanguage",
			"ailanguage",
			apiErrorCoverageDefaultVersion,
			"Endpoint",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("ailanguage", "EvaluationResult"): reviewedException(
			"ailanguage",
			"ailanguage",
			apiErrorCoverageDefaultVersion,
			"EvaluationResult",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("ailanguage", "Model"): reviewedException(
			"ailanguage",
			"ailanguage",
			apiErrorCoverageDefaultVersion,
			"Model",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("ailanguage", "ModelType"): reviewedException(
			"ailanguage",
			"ailanguage",
			apiErrorCoverageDefaultVersion,
			"ModelType",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("ailanguage", "WorkRequest"): reviewedException(
			"ailanguage",
			"ailanguage",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("ailanguage", "WorkRequestError"): reviewedException(
			"ailanguage",
			"ailanguage",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("ailanguage", "WorkRequestLog"): reviewedException(
			"ailanguage",
			"ailanguage",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aispeech", "TranscriptionTask"): reviewedException(
			"aispeech",
			"aispeech",
			apiErrorCoverageDefaultVersion,
			"TranscriptionTask",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aivision", "DocumentJob"): reviewedException(
			"aivision",
			"aivision",
			apiErrorCoverageDefaultVersion,
			"DocumentJob",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aivision", "ImageJob"): reviewedException(
			"aivision",
			"aivision",
			apiErrorCoverageDefaultVersion,
			"ImageJob",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aivision", "Model"): reviewedException(
			"aivision",
			"aivision",
			apiErrorCoverageDefaultVersion,
			"Model",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aivision", "WorkRequest"): reviewedException(
			"aivision",
			"aivision",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aivision", "WorkRequestError"): reviewedException(
			"aivision",
			"aivision",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("aivision", "WorkRequestLog"): reviewedException(
			"aivision",
			"aivision",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("analytics", "PrivateAccessChannel"): reviewedException(
			"analytics",
			"analytics",
			apiErrorCoverageDefaultVersion,
			"PrivateAccessChannel",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("analytics", "VanityUrl"): reviewedException(
			"analytics",
			"analytics",
			apiErrorCoverageDefaultVersion,
			"VanityUrl",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("analytics", "WorkRequest"): reviewedException(
			"analytics",
			"analytics",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("analytics", "WorkRequestError"): reviewedException(
			"analytics",
			"analytics",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("analytics", "WorkRequestLog"): reviewedException(
			"analytics",
			"analytics",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apiplatform", "WorkRequest"): reviewedException(
			"apiplatform",
			"apiplatform",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apiplatform", "WorkRequestError"): reviewedException(
			"apiplatform",
			"apiplatform",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apiplatform", "WorkRequestLog"): reviewedException(
			"apiplatform",
			"apiplatform",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apmcontrolplane", "ApmDomainWorkRequest"): reviewedException(
			"apmcontrolplane",
			"apmcontrolplane",
			apiErrorCoverageDefaultVersion,
			"ApmDomainWorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apmcontrolplane", "DataKey"): reviewedException(
			"apmcontrolplane",
			"apmcontrolplane",
			apiErrorCoverageDefaultVersion,
			"DataKey",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apmcontrolplane", "WorkRequest"): reviewedException(
			"apmcontrolplane",
			"apmcontrolplane",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apmcontrolplane", "WorkRequestError"): reviewedException(
			"apmcontrolplane",
			"apmcontrolplane",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("apmcontrolplane", "WorkRequestLog"): reviewedException(
			"apmcontrolplane",
			"apmcontrolplane",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("bds", "AutoScalingConfiguration"): reviewedException(
			"bds",
			"bds",
			apiErrorCoverageDefaultVersion,
			"AutoScalingConfiguration",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("bds", "BdsApiKey"): reviewedException(
			"bds",
			"bds",
			apiErrorCoverageDefaultVersion,
			"BdsApiKey",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("bds", "BdsMetastoreConfiguration"): reviewedException(
			"bds",
			"bds",
			apiErrorCoverageDefaultVersion,
			"BdsMetastoreConfiguration",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("bds", "OsPatch"): reviewedException(
			"bds",
			"bds",
			apiErrorCoverageDefaultVersion,
			"OsPatch",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("bds", "OsPatchDetail"): reviewedException(
			"bds",
			"bds",
			apiErrorCoverageDefaultVersion,
			"OsPatchDetail",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("bds", "Patch"): reviewedException(
			"bds",
			"bds",
			apiErrorCoverageDefaultVersion,
			"Patch",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("bds", "PatchHistory"): reviewedException(
			"bds",
			"bds",
			apiErrorCoverageDefaultVersion,
			"PatchHistory",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("bds", "WorkRequest"): reviewedException(
			"bds",
			"bds",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("bds", "WorkRequestError"): reviewedException(
			"bds",
			"bds",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("bds", "WorkRequestLog"): reviewedException(
			"bds",
			"bds",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("budget", "AlertRule"): reviewedException(
			"budget",
			"budget",
			apiErrorCoverageDefaultVersion,
			"AlertRule",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("budget", "CostAlertSubscription"): reviewedException(
			"budget",
			"budget",
			apiErrorCoverageDefaultVersion,
			"CostAlertSubscription",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("budget", "CostAnomalyEvent"): reviewedException(
			"budget",
			"budget",
			apiErrorCoverageDefaultVersion,
			"CostAnomalyEvent",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("budget", "CostAnomalyMonitor"): reviewedException(
			"budget",
			"budget",
			apiErrorCoverageDefaultVersion,
			"CostAnomalyMonitor",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("clusterplacementgroups", "WorkRequest"): reviewedException(
			"clusterplacementgroups",
			"clusterplacementgroups",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("clusterplacementgroups", "WorkRequestError"): reviewedException(
			"clusterplacementgroups",
			"clusterplacementgroups",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("clusterplacementgroups", "WorkRequestLog"): reviewedException(
			"clusterplacementgroups",
			"clusterplacementgroups",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("databasetools", "DatabaseToolsEndpointService"): reviewedException(
			"databasetools",
			"databasetools",
			apiErrorCoverageDefaultVersion,
			"DatabaseToolsEndpointService",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("databasetools", "DatabaseToolsPrivateEndpoint"): reviewedException(
			"databasetools",
			"databasetools",
			apiErrorCoverageDefaultVersion,
			"DatabaseToolsPrivateEndpoint",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("databasetools", "WorkRequest"): reviewedException(
			"databasetools",
			"databasetools",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("databasetools", "WorkRequestError"): reviewedException(
			"databasetools",
			"databasetools",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("databasetools", "WorkRequestLog"): reviewedException(
			"databasetools",
			"databasetools",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datalabelingservice", "AnnotationFormat"): reviewedException(
			"datalabelingservice",
			"datalabelingservice",
			apiErrorCoverageDefaultVersion,
			"AnnotationFormat",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datalabelingservice", "WorkRequest"): reviewedException(
			"datalabelingservice",
			"datalabelingservice",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datalabelingservice", "WorkRequestError"): reviewedException(
			"datalabelingservice",
			"datalabelingservice",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datalabelingservice", "WorkRequestLog"): reviewedException(
			"datalabelingservice",
			"datalabelingservice",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "DataSciencePrivateEndpoint"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"DataSciencePrivateEndpoint",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "FastLaunchJobConfig"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"FastLaunchJobConfig",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "Job"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"Job",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "JobArtifact"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"JobArtifact",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "JobArtifactContent"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"JobArtifactContent",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "JobRun"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"JobRun",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "JobShape"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"JobShape",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "Model"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"Model",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "ModelArtifact"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"ModelArtifact",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "ModelArtifactContent"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"ModelArtifactContent",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "ModelDeployment"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"ModelDeployment",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "ModelDeploymentShape"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"ModelDeploymentShape",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "ModelProvenance"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"ModelProvenance",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "ModelVersionSet"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"ModelVersionSet",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "NotebookSession"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"NotebookSession",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "NotebookSessionShape"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"NotebookSessionShape",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "Pipeline"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"Pipeline",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "PipelineRun"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"PipelineRun",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "StepArtifact"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"StepArtifact",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "StepArtifactContent"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"StepArtifactContent",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "WorkRequest"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"WorkRequest",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "WorkRequestError"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"WorkRequestError",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("datascience", "WorkRequestLog"): reviewedException(
			"datascience",
			"datascience",
			apiErrorCoverageDefaultVersion,
			"WorkRequestLog",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
		resourceKey("dashboardservice", "Dashboard"): reviewedException(
			"dashboardservice",
			"dashboardservice",
			apiErrorCoverageDefaultVersion,
			"Dashboard",
			"`services.yaml` keeps this subresource out of the active controller-backed surface with controller.strategy=none and serviceManager.strategy=none.",
		),
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

func mergeReviewedRegistrationSets(sets ...map[string]APIErrorCoverageRegistration) map[string]APIErrorCoverageRegistration {
	merged := make(map[string]APIErrorCoverageRegistration)
	for _, set := range sets {
		for key, registration := range set {
			if _, exists := merged[key]; exists {
				panic(fmt.Sprintf("duplicate reviewed registration %q", key))
			}
			merged[key] = registration
		}
	}
	return merged
}

func reviewedRegistrationSet(
	service string,
	group string,
	version string,
	family APIErrorCoverageFamily,
	deleteNotFoundSemantics string,
	retryableConflictSemantics string,
	deviation string,
	kinds ...string,
) map[string]APIErrorCoverageRegistration {
	registrations := make(map[string]APIErrorCoverageRegistration, len(kinds))
	for _, kind := range kinds {
		registrations[resourceKey(service, kind)] = reviewedRegistration(
			service,
			group,
			version,
			kind,
			family,
			deleteNotFoundSemantics,
			retryableConflictSemantics,
			deviation,
		)
	}
	return registrations
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
