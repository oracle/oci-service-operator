package errortest

import (
	"strings"
	"testing"
)

func TestCheckedInAPIErrorCoverageInventoryIncludesSelectedKindsAndExplicitExceptions(t *testing.T) {
	t.Parallel()

	inventory, err := LoadCheckedInAPIErrorCoverageInventory()
	if err != nil {
		t.Fatalf("LoadCheckedInAPIErrorCoverageInventory() error = %v", err)
	}

	byKey := inventoryByKey(inventory)
	if got, want := len(inventory), 105; got != want {
		t.Fatalf("len(inventory) = %d, want %d", got, want)
	}
	if got, want := countRegistrations(inventory), 34; got != want {
		t.Fatalf("registration inventory count = %d, want %d", got, want)
	}
	if got, want := countExceptions(inventory), 71; got != want {
		t.Fatalf("exception inventory count = %d, want %d", got, want)
	}

	assertInventorySelectionSource(t, byKey, "aidocument/Project", "selection.includeKinds")
	assertInventorySelectionSource(t, byKey, "ailanguage/Project", "selection.includeKinds")
	assertInventorySelectionSource(t, byKey, "aivision/Project", "selection.includeKinds")
	assertInventorySelectionSource(t, byKey, "analytics/AnalyticsInstance", "selection.includeKinds")
	assertInventorySelectionSource(t, byKey, "bds/BdsInstance", "selection.includeKinds")
	assertInventorySelectionSource(t, byKey, "containerengine/Cluster", "selection.includeKinds")
	assertInventorySelectionSource(t, byKey, "containerengine/NodePool", "selection.includeKinds")
	assertInventorySelectionSource(t, byKey, "core/Drg", "packageSplits[core-network].includeKinds")
	assertInventorySelectionSource(t, byKey, "databasetools/DatabaseToolsConnection", "selection.includeKinds")
	assertInventorySelectionSource(t, byKey, "datascience/Project", "selection.includeKinds")
	assertInventorySelectionSource(t, byKey, "dataflow/Application", "selection.includeKinds")
	assertInventorySelectionSource(t, byKey, "functions/Application", "selection.includeKinds")
	assertInventorySelectionSource(t, byKey, "queue/Queue", "selection.includeKinds")

	assertInventoryRegistration(t, byKey, "aidocument/Project")
	assertInventoryException(t, byKey, "aidocument/WorkRequest", `controller.strategy="none"`)
	assertInventoryRegistration(t, byKey, "ailanguage/Project")
	assertInventoryException(t, byKey, "ailanguage/Endpoint", `controller.strategy="none"`)
	assertInventoryRegistration(t, byKey, "aivision/Project")
	assertInventoryException(t, byKey, "aivision/WorkRequest", `controller.strategy="none"`)
	assertInventoryRegistration(t, byKey, "analytics/AnalyticsInstance")
	assertInventoryException(t, byKey, "analytics/PrivateAccessChannel", `controller.strategy="none"`)
	assertInventoryRegistration(t, byKey, "bds/BdsInstance")
	assertInventoryException(t, byKey, "bds/WorkRequest", `controller.strategy="none"`)
	assertInventoryRegistration(t, byKey, "databasetools/DatabaseToolsConnection")
	assertInventoryException(t, byKey, "databasetools/WorkRequest", `controller.strategy="none"`)
	assertInventoryRegistration(t, byKey, "datascience/Project")
	assertInventoryException(t, byKey, "datascience/WorkRequest", `controller.strategy="none"`)
	assertInventoryRegistration(t, byKey, "keymanagement/Vault")
	assertInventoryException(t, byKey, "keymanagement/Key", `controller.strategy="none"`)
	assertInventoryException(t, byKey, "opensearch/WorkRequestLog", `controller.strategy="none"`)
}

func TestReviewedAPIErrorCoverageRegistryMatchesCheckedInInventory(t *testing.T) {
	t.Parallel()

	if err := ValidateCheckedInAPIErrorCoverageRegistry(); err != nil {
		t.Fatalf("ValidateCheckedInAPIErrorCoverageRegistry() error = %v", err)
	}
}

func TestReviewedAPIErrorCoverageRegistryRepresentativeMappings(t *testing.T) {
	t.Parallel()

	assertReviewedFamily(t, "aidocument/Project", APIErrorCoverageFamilyGeneratedRuntimePlain)
	assertReviewedFamily(t, "ailanguage/Project", APIErrorCoverageFamilyGeneratedRuntimePlain)
	assertReviewedFamily(t, "aivision/Project", APIErrorCoverageFamilyGeneratedRuntimePlain)
	assertReviewedFamily(t, "analytics/AnalyticsInstance", APIErrorCoverageFamilyGeneratedRuntimePlain)
	assertReviewedFamily(t, "bds/BdsInstance", APIErrorCoverageFamilyGeneratedRuntimePlain)
	assertReviewedFamily(t, "containerengine/Cluster", APIErrorCoverageFamilyGeneratedRuntimeFollowUp)
	assertReviewedFamily(t, "containerengine/NodePool", APIErrorCoverageFamilyGeneratedRuntimePlain)
	assertReviewedFamily(t, "containerinstances/ContainerInstance", APIErrorCoverageFamilyLegacyAdapter)
	assertReviewedFamily(t, "core/Vcn", APIErrorCoverageFamilyManualRuntime)
	assertReviewedFamily(t, "databasetools/DatabaseToolsConnection", APIErrorCoverageFamilyGeneratedRuntimePlain)
	assertReviewedFamily(t, "datascience/Project", APIErrorCoverageFamilyGeneratedRuntimePlain)
	assertReviewedFamily(t, "dataflow/Application", APIErrorCoverageFamilyManualRuntime)
	assertReviewedFamily(t, "functions/Application", APIErrorCoverageFamilyLegacyAdapter)
	assertReviewedFamily(t, "functions/Function", APIErrorCoverageFamilyLegacyAdapter)
	assertReviewedFamily(t, "identity/Compartment", APIErrorCoverageFamilyLegacyAdapter)
	assertReviewedFamily(t, "keymanagement/Vault", APIErrorCoverageFamilyLegacyAdapter)
	assertReviewedFamily(t, "nosql/Table", APIErrorCoverageFamilyLegacyAdapter)
	assertReviewedFamily(t, "objectstorage/Bucket", APIErrorCoverageFamilyGeneratedRuntimePlain)
	assertReviewedFamily(t, "psql/DbSystem", APIErrorCoverageFamilyLegacyAdapter)
	assertReviewedFamily(t, "queue/Queue", APIErrorCoverageFamilyGeneratedRuntimeWorkRequest)
	assertReviewedFamily(t, "redis/RedisCluster", APIErrorCoverageFamilyGeneratedRuntimeWorkRequest)
	assertReviewedFamily(t, "streaming/Stream", APIErrorCoverageFamilyGeneratedRuntimeFollowUp)

	assertReviewedException(t, "aidocument/WorkRequest", "strategy=none")
	assertReviewedException(t, "ailanguage/WorkRequest", "strategy=none")
	assertReviewedException(t, "aivision/WorkRequest", "strategy=none")
	assertReviewedException(t, "analytics/WorkRequest", "strategy=none")
	assertReviewedException(t, "bds/WorkRequest", "strategy=none")
	assertReviewedException(t, "databasetools/WorkRequest", "strategy=none")
	assertReviewedException(t, "datascience/WorkRequest", "strategy=none")
	assertReviewedException(t, "keymanagement/Key", "strategy=none")
	assertReviewedException(t, "opensearch/WorkRequest", "strategy=none")
}

func TestReviewedAPIErrorCoverageRegistryAsyncDeviationsRemainExplicit(t *testing.T) {
	t.Parallel()

	assertReviewedDeviation(t, "opensearch/OpensearchCluster", "read-after-write")
	assertReviewedDeviation(t, "psql/DbSystem", "readback adapter")
	assertReviewedDeviation(t, "queue/Queue", "work-request")
	assertReviewedDeviation(t, "redis/RedisCluster", "delete guard")
	assertReviewedDeviation(t, "streaming/Stream", "WaitForUpdatedState")
}

func TestReviewedAPIErrorCoverageRegistryNodePoolPlainSemantics(t *testing.T) {
	t.Parallel()

	assertReviewedFamily(t, "containerengine/NodePool", APIErrorCoverageFamilyGeneratedRuntimePlain)
	assertReviewedSemantics(t, "containerengine/NodePool", deleteNotFoundGeneratedRuntime, retryableConflictGeneratedRuntime)
	assertReviewedDeviation(t, "containerengine/NodePool", "request-body shaping")
}

func TestReviewedAPIErrorCoverageRegistryRedisWorkRequestSemantics(t *testing.T) {
	t.Parallel()

	assertReviewedDeleteSemantics(t, "redis/RedisCluster", deleteNotFoundReadback)
	assertReviewedRetryableConflictContains(t, "redis/RedisCluster", "work-request")
	assertReviewedRetryableConflictContains(t, "redis/RedisCluster", "reread live RedisCluster state")
	assertReviewedDeviation(t, "redis/RedisCluster", "delete guard")
}

func TestReviewedAPIErrorCoverageRegistrySplitCoreDeleteSemantics(t *testing.T) {
	t.Parallel()

	for _, key := range []string{
		"core/InternetGateway",
		"core/NatGateway",
		"core/NetworkSecurityGroup",
		"core/ServiceGateway",
		"core/Vcn",
	} {
		assertReviewedSemantics(t, key, deleteNotFoundGeneratedRuntime, retryableConflictGeneratedRuntime)
		assertReviewedDeviation(t, key, "Delete delegates to generatedruntime confirm-delete semantics")
	}

	for _, key := range []string{
		"core/RouteTable",
		"core/SecurityList",
		"core/Subnet",
	} {
		assertReviewedSemantics(t, key, deleteNotFoundManualRuntime, retryableConflictManualRuntime)
	}
}

func TestReviewedAPIErrorCoverageRegistryLegacyAdapterSemantics(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		key                string
		wantDelete         string
		wantRetrySubstring string
		wantDeviation      string
	}{
		{
			key:                "containerinstances/ContainerInstance",
			wantDelete:         deleteNotFoundManualRuntime,
			wantRetrySubstring: "helper-specific rereads or adapter state checks",
			wantDeviation:      "create-or-bind lookup",
		},
		{
			key:                "functions/Application",
			wantDelete:         deleteNotFoundManualRuntime,
			wantRetrySubstring: "helper-specific rereads or adapter state checks",
			wantDeviation:      "delete helpers",
		},
		{
			key:                "functions/Function",
			wantDelete:         deleteNotFoundManualRuntime,
			wantRetrySubstring: "helper-specific rereads or adapter state checks",
			wantDeviation:      "endpoint-secret side effects",
		},
		{
			key:                "identity/Compartment",
			wantDelete:         deleteNotFoundPendingDeletion,
			wantRetrySubstring: "helper-specific rereads or adapter state checks",
			wantDeviation:      "Orphan-delete helper",
		},
		{
			key:                "keymanagement/Vault",
			wantDelete:         deleteNotFoundPendingDeletion,
			wantRetrySubstring: "pending-deletion lifecycle and scheduled-delete handling",
			wantDeviation:      "schedule deletion windows",
		},
		{
			key:                "nosql/Table",
			wantDelete:         deleteNotFoundReadback,
			wantRetrySubstring: "helper-specific rereads or adapter state checks",
			wantDeviation:      "errTableNotFound",
		},
		{
			key:                "psql/DbSystem",
			wantDelete:         deleteNotFoundReadback,
			wantRetrySubstring: "helper-specific rereads or adapter state checks",
			wantDeviation:      "readback adapter",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()

			assertReviewedDeleteSemantics(t, tc.key, tc.wantDelete)
			assertReviewedRetryableConflictContains(t, tc.key, tc.wantRetrySubstring)
			assertReviewedDeviation(t, tc.key, tc.wantDeviation)
		})
	}
}

func inventoryByKey(inventory []APIErrorCoverageInventoryItem) map[string]APIErrorCoverageInventoryItem {
	items := make(map[string]APIErrorCoverageInventoryItem, len(inventory))
	for _, item := range inventory {
		items[item.Key()] = item
	}
	return items
}

func countRegistrations(inventory []APIErrorCoverageInventoryItem) int {
	total := 0
	for _, item := range inventory {
		if item.RequiresRegistration() {
			total++
		}
	}
	return total
}

func countExceptions(inventory []APIErrorCoverageInventoryItem) int {
	total := 0
	for _, item := range inventory {
		if item.RequiresException() {
			total++
		}
	}
	return total
}

func assertInventorySelectionSource(
	t *testing.T,
	byKey map[string]APIErrorCoverageInventoryItem,
	key string,
	wantSource string,
) {
	t.Helper()

	item, ok := byKey[key]
	if !ok {
		t.Fatalf("inventory item %q was not found", key)
	}
	for _, source := range item.SelectionSources {
		if source == wantSource {
			return
		}
	}
	t.Fatalf("%s selectionSources = %v, want %q", key, item.SelectionSources, wantSource)
}

func assertInventoryRegistration(
	t *testing.T,
	byKey map[string]APIErrorCoverageInventoryItem,
	key string,
) {
	t.Helper()

	item, ok := byKey[key]
	if !ok {
		t.Fatalf("inventory item %q was not found", key)
	}
	if !item.RequiresRegistration() {
		t.Fatalf("%s RequiresRegistration() = false, selectionSources=%v exception=%q", key, item.SelectionSources, item.ExceptionReason)
	}
}

func assertInventoryException(
	t *testing.T,
	byKey map[string]APIErrorCoverageInventoryItem,
	key string,
	wantSubstring string,
) {
	t.Helper()

	item, ok := byKey[key]
	if !ok {
		t.Fatalf("inventory item %q was not found", key)
	}
	if !item.RequiresException() {
		t.Fatalf("%s RequiresException() = false, selectionSources=%v", key, item.SelectionSources)
	}
	if !strings.Contains(item.ExceptionReason, wantSubstring) {
		t.Fatalf("%s exceptionReason = %q, want substring %q", key, item.ExceptionReason, wantSubstring)
	}
}

func assertReviewedFamily(t *testing.T, key string, want APIErrorCoverageFamily) {
	t.Helper()

	registration, ok := ReviewedAPIErrorCoverageRegistry.Registrations[key]
	if !ok {
		t.Fatalf("reviewed registration %q was not found", key)
	}
	if registration.Family != want {
		t.Fatalf("%s family = %q, want %q", key, registration.Family, want)
	}
	if strings.TrimSpace(registration.DeleteNotFoundSemantics) == "" {
		t.Fatalf("%s deleteNotFoundSemantics is empty", key)
	}
	if strings.TrimSpace(registration.RetryableConflictSemantics) == "" {
		t.Fatalf("%s retryableConflictSemantics is empty", key)
	}
}

func assertReviewedSemantics(
	t *testing.T,
	key string,
	wantDeleteNotFound string,
	wantRetryableConflict string,
) {
	t.Helper()

	registration, ok := ReviewedAPIErrorCoverageRegistry.Registrations[key]
	if !ok {
		t.Fatalf("reviewed registration %q was not found", key)
	}
	if registration.DeleteNotFoundSemantics != wantDeleteNotFound {
		t.Fatalf("%s deleteNotFoundSemantics = %q, want %q", key, registration.DeleteNotFoundSemantics, wantDeleteNotFound)
	}
	if registration.RetryableConflictSemantics != wantRetryableConflict {
		t.Fatalf("%s retryableConflictSemantics = %q, want %q", key, registration.RetryableConflictSemantics, wantRetryableConflict)
	}
}

func assertReviewedDeleteSemantics(t *testing.T, key string, wantDeleteNotFound string) {
	t.Helper()

	registration, ok := ReviewedAPIErrorCoverageRegistry.Registrations[key]
	if !ok {
		t.Fatalf("reviewed registration %q was not found", key)
	}
	if registration.DeleteNotFoundSemantics != wantDeleteNotFound {
		t.Fatalf("%s deleteNotFoundSemantics = %q, want %q", key, registration.DeleteNotFoundSemantics, wantDeleteNotFound)
	}
}

func assertReviewedRetryableConflictContains(t *testing.T, key string, wantSubstring string) {
	t.Helper()

	registration, ok := ReviewedAPIErrorCoverageRegistry.Registrations[key]
	if !ok {
		t.Fatalf("reviewed registration %q was not found", key)
	}
	if !strings.Contains(registration.RetryableConflictSemantics, wantSubstring) {
		t.Fatalf("%s retryableConflictSemantics = %q, want substring %q", key, registration.RetryableConflictSemantics, wantSubstring)
	}
}

func assertReviewedDeviation(t *testing.T, key string, wantSubstring string) {
	t.Helper()

	registration, ok := ReviewedAPIErrorCoverageRegistry.Registrations[key]
	if !ok {
		t.Fatalf("reviewed registration %q was not found", key)
	}
	if !strings.Contains(registration.Deviation, wantSubstring) {
		t.Fatalf("%s deviation = %q, want substring %q", key, registration.Deviation, wantSubstring)
	}
}

func assertReviewedException(t *testing.T, key string, wantSubstring string) {
	t.Helper()

	exception, ok := ReviewedAPIErrorCoverageRegistry.Exceptions[key]
	if !ok {
		t.Fatalf("reviewed exception %q was not found", key)
	}
	if !strings.Contains(exception.Reason, wantSubstring) {
		t.Fatalf("%s reason = %q, want substring %q", key, exception.Reason, wantSubstring)
	}
}
