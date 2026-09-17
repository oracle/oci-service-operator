/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type commandResponse struct {
	contains []string
	output   string
	err      error
}

type fakeCommandRunner struct {
	t         *testing.T
	mu        sync.Mutex
	responses []commandResponse
	calls     []string
}

func (f *fakeCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	call := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call)
	if len(f.responses) == 0 {
		f.t.Fatalf("unexpected command: %s", call)
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	for _, expected := range response.contains {
		if !strings.Contains(call, expected) {
			f.t.Fatalf("command %q does not contain %q", call, expected)
		}
	}
	return []byte(response.output), response.err
}

func TestRunCreateUpdateDeleteLifecycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "scenario.yaml", `version: 1
name: budget-crud
service: budget
namespace: osok-e2e
timeout: 1s
pollInterval: 1ms
dependencies:
  - dependency.yaml
create: create.yaml
update: update.yaml
ready:
  conditionTypes: [Active]
  lifecycleStates: [ACTIVE]
  requireOCID: true
  observeGeneration: true
`)
	writeTestFile(t, root, "dependency.yaml", `apiVersion: v1
kind: Secret
metadata:
  name: test-${OSOK_E2E_SUFFIX}
stringData:
  password: ${TEST_PASSWORD}
`)
	writeTestFile(t, root, "create.yaml", testBudgetManifest("initial"))
	writeTestFile(t, root, "update.yaml", testBudgetManifest("updated"))

	createReady := resourceJSON(1, 1, "Active", "True", "ACTIVE", "ocid1.budget.oc1..created")
	updateReady := resourceJSON(2, 2, "Active", "True", "ACTIVE", "ocid1.budget.oc1..created")
	fake := &fakeCommandRunner{t: t, responses: []commandResponse{
		{contains: []string{"apply", "dependency-01.yaml"}, output: "secret configured"},
		{contains: []string{"apply", "create.yaml"}, output: "budget created"},
		{contains: []string{"get", "budget.budget.oracle.com", "budget-demo"}, output: createReady},
		{contains: []string{"apply", "update.yaml"}, output: "budget configured"},
		{contains: []string{"get", "budget.budget.oracle.com", "budget-demo"}, output: updateReady},
		{contains: []string{"delete", "budget.budget.oracle.com", "--wait=false"}, output: "budget deleted"},
		{contains: []string{"get", "budget.budget.oracle.com"}, output: "Error from server (NotFound)", err: errors.New("exit status 1")},
		{contains: []string{"delete", "dependency-01.yaml", "--wait=true"}, output: "secret deleted"},
	}}
	artifacts := filepath.Join(root, "artifacts")
	result, err := Run(context.Background(), RunOptions{
		ScenarioPath:  filepath.Join(root, "scenario.yaml"),
		ArtifactsDir:  artifacts,
		Kubeconfig:    filepath.Join(root, "kubeconfig"),
		CommandRunner: fake,
		Variables: map[string]string{
			"OSOK_E2E_SUFFIX":    "fixed",
			"TEST_PASSWORD":      "not-committed",
			"OCI_COMPARTMENT_ID": "ocid1.compartment.oc1..test",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "passed" {
		t.Fatalf("result status = %q", result.Status)
	}
	if len(result.Phases) != 8 {
		t.Fatalf("phase count = %d, want 8", len(result.Phases))
	}
	if len(fake.responses) != 0 {
		t.Fatalf("unused fake command responses = %d", len(fake.responses))
	}
	resultContent, err := os.ReadFile(filepath.Join(artifacts, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(resultContent), `"status": "passed"`) {
		t.Fatalf("result.json = %s", resultContent)
	}
	rendered, err := os.ReadFile(filepath.Join(artifacts, "rendered", "create.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(rendered), "${") || !strings.Contains(string(rendered), "ocid1.compartment.oc1..test") {
		t.Fatalf("rendered create manifest = %s", rendered)
	}
}

func TestRunFailureCleansCreatedResourceAndDependencies(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "scenario.yaml", `version: 1
name: failed-budget
service: budget
timeout: 1s
pollInterval: 1ms
dependencies: [dependency.yaml]
create: create.yaml
`)
	writeTestFile(t, root, "dependency.yaml", "apiVersion: v1\nkind: Secret\nmetadata:\n  name: helper\n")
	writeTestFile(t, root, "create.yaml", testBudgetManifest("initial"))
	fake := &fakeCommandRunner{t: t, responses: []commandResponse{
		{contains: []string{"apply", "dependency-01.yaml"}, output: "configured"},
		{contains: []string{"apply", "create.yaml"}, output: "created"},
		{contains: []string{"get", "budget.budget.oracle.com"}, output: resourceJSON(1, 1, "Failed", "False", "FAILED", "")},
		{contains: []string{"delete", "budget.budget.oracle.com"}, output: "deleted"},
		{contains: []string{"get", "budget.budget.oracle.com"}, output: "NotFound", err: errors.New("exit 1")},
		{contains: []string{"delete", "dependency-01.yaml"}, output: "deleted"},
	}}
	result, err := Run(context.Background(), RunOptions{
		ScenarioPath:  filepath.Join(root, "scenario.yaml"),
		ArtifactsDir:  filepath.Join(root, "artifacts"),
		CommandRunner: fake,
		Variables: map[string]string{
			"OSOK_E2E_SUFFIX":    "fixed",
			"OCI_COMPARTMENT_ID": "ocid1.compartment.oc1..test",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "Failed=False") {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("result status = %q", result.Status)
	}
	if result.CleanupError != "" {
		t.Fatalf("cleanup error = %q", result.CleanupError)
	}
	if len(fake.responses) != 0 {
		t.Fatalf("unused fake command responses = %d", len(fake.responses))
	}
}

func TestRunVerifiesRelatedObjectLifecycle(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "scenario.yaml", `version: 1
name: related-secret
service: budget
timeout: 1s
pollInterval: 1ms
create: create.yaml
relatedObjects:
  - apiVersion: v1
    kind: Secret
    requiredDataKeys: [endpoint]
    deleteWithResource: true
`)
	writeTestFile(t, root, "create.yaml", testBudgetManifest("initial"))
	fake := &fakeCommandRunner{t: t, responses: []commandResponse{
		{contains: []string{"apply", "create.yaml"}, output: "created"},
		{contains: []string{"get", "budget.budget.oracle.com"}, output: resourceJSON(1, 1, "Active", "True", "ACTIVE", "ocid1.budget.oc1..created")},
		{contains: []string{"get", "secret", "budget-demo"}, output: `{"apiVersion":"v1","kind":"Secret","metadata":{"name":"budget-demo","namespace":"default"},"data":{"endpoint":"aHR0cHM6Ly9leGFtcGxl"}}`},
		{contains: []string{"delete", "budget.budget.oracle.com"}, output: "deleted"},
		{contains: []string{"get", "budget.budget.oracle.com"}, output: "NotFound", err: errors.New("exit 1")},
		{contains: []string{"get", "secret", "budget-demo"}, output: "NotFound", err: errors.New("exit 1")},
	}}
	result, err := Run(context.Background(), RunOptions{
		ScenarioPath:  filepath.Join(root, "scenario.yaml"),
		ArtifactsDir:  filepath.Join(root, "artifacts"),
		CommandRunner: fake,
		Variables: map[string]string{
			"OSOK_E2E_SUFFIX":    "fixed",
			"OCI_COMPARTMENT_ID": "ocid1.compartment.oc1..test",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "passed" {
		t.Fatalf("result status = %q", result.Status)
	}
	if len(result.Phases) != 6 {
		t.Fatalf("phase count = %d, want 6", len(result.Phases))
	}
	if got := result.Phases[2].Name; got != "verify_related_after_create" {
		t.Fatalf("phase[2] = %q", got)
	}
	if got := result.Phases[5].Name; got != "verify_related_deletion" {
		t.Fatalf("phase[5] = %q", got)
	}
}

func TestLoadScenarioRejectsUnknownFieldAndEscapingPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "create.yaml", testBudgetManifest("initial"))
	writeTestFile(t, root, "unknown.yaml", "version: 1\nname: x\nservice: budget\ncreate: create.yaml\nunknown: true\n")
	if _, err := LoadScenario(filepath.Join(root, "unknown.yaml")); err == nil || !strings.Contains(err.Error(), "field unknown not found") {
		t.Fatalf("LoadScenario(unknown) error = %v", err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside.yaml")
	if err := os.WriteFile(outside, []byte(testBudgetManifest("initial")), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(outside) })
	writeTestFile(t, root, "escape.yaml", "version: 1\nname: x\nservice: budget\ncreate: ../outside.yaml\n")
	if _, err := LoadScenario(filepath.Join(root, "escape.yaml")); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("LoadScenario(escape) error = %v", err)
	}
}

func TestLoadScenarioRejectsInvalidRelatedObject(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "create.yaml", testBudgetManifest("initial"))
	writeTestFile(t, root, "scenario.yaml", `version: 1
name: invalid-related-object
service: budget
create: create.yaml
relatedObjects:
  - apiVersion: invalid/version/extra
    kind: Secret
`)
	if _, err := LoadScenario(filepath.Join(root, "scenario.yaml")); err == nil || !strings.Contains(err.Error(), "apiVersion") {
		t.Fatalf("LoadScenario(invalid related object) error = %v", err)
	}
}

func TestRunRejectsUnsetVariablesAndServiceMismatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "scenario.yaml", "version: 1\nname: x\nservice: budget\ncreate: create.yaml\n")
	writeTestFile(t, root, "create.yaml", testBudgetManifest("initial"))
	if _, err := Run(context.Background(), RunOptions{
		ScenarioPath:    filepath.Join(root, "scenario.yaml"),
		ArtifactsDir:    filepath.Join(root, "mismatch"),
		ExpectedService: "streaming",
	}); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("Run(service mismatch) error = %v", err)
	}
	missingVariable := unusedEnvironmentVariable()
	writeTestFile(t, root, "create.yaml", strings.ReplaceAll(
		testBudgetManifest("initial"),
		"${OCI_COMPARTMENT_ID}",
		"${"+missingVariable+"}",
	))
	fake := &fakeCommandRunner{t: t}
	if _, err := Run(context.Background(), RunOptions{
		ScenarioPath:  filepath.Join(root, "scenario.yaml"),
		ArtifactsDir:  filepath.Join(root, "missing"),
		CommandRunner: fake,
		Variables:     map[string]string{"OSOK_E2E_SUFFIX": "fixed"},
	}); err == nil || !strings.Contains(err.Error(), missingVariable) {
		t.Fatalf("Run(missing variable) error = %v", err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("missing-variable validation executed commands: %v", fake.calls)
	}
}

func TestPrepareRunOptionsCreatesCollisionResistantDefaults(t *testing.T) {
	t.Parallel()

	const runs = 16
	root := t.TempDir()
	fixedNow := time.Date(2026, time.September, 17, 12, 34, 56, 0, time.UTC)
	type preparedResult struct {
		options RunOptions
		err     error
	}
	results := make(chan preparedResult, runs)
	var group sync.WaitGroup
	for range runs {
		group.Add(1)
		go func() {
			defer group.Done()
			options, err := prepareRunOptions(RunOptions{
				Now:       func() time.Time { return fixedNow },
				Variables: map[string]string{"OSOK_E2E_SUFFIX": ""},
			}, "same-scenario", root)
			results <- preparedResult{options: options, err: err}
		}()
	}
	group.Wait()
	close(results)

	artifactRoot := filepath.Join(root, "osok-e2e", "same-scenario")
	artifactDirectories := map[string]struct{}{}
	resourceSuffixes := map[string]struct{}{}
	for prepared := range results {
		if prepared.err != nil {
			t.Fatal(prepared.err)
		}
		if !strings.HasPrefix(prepared.options.ArtifactsDir, artifactRoot+string(filepath.Separator)) {
			t.Fatalf("default artifacts directory = %q, want child of %q", prepared.options.ArtifactsDir, artifactRoot)
		}
		if _, exists := artifactDirectories[prepared.options.ArtifactsDir]; exists {
			t.Fatalf("duplicate artifacts directory: %s", prepared.options.ArtifactsDir)
		}
		artifactDirectories[prepared.options.ArtifactsDir] = struct{}{}

		suffix := prepared.options.Variables["OSOK_E2E_SUFFIX"]
		if !strings.HasPrefix(suffix, "20260917-123456-") {
			t.Fatalf("generated suffix = %q", suffix)
		}
		if len(suffix) != len("20260917-123456-")+12 {
			t.Fatalf("generated suffix length = %d, want %d", len(suffix), len("20260917-123456-")+12)
		}
		if _, exists := resourceSuffixes[suffix]; exists {
			t.Fatalf("duplicate resource suffix: %s", suffix)
		}
		resourceSuffixes[suffix] = struct{}{}
	}
	if len(artifactDirectories) != runs || len(resourceSuffixes) != runs {
		t.Fatalf("unique defaults = artifacts:%d suffixes:%d, want %d", len(artifactDirectories), len(resourceSuffixes), runs)
	}
}

func TestPrepareRunOptionsPreservesExplicitOverrides(t *testing.T) {
	t.Parallel()

	artifactsDir := filepath.Join(t.TempDir(), "operator-artifacts")
	prepared, err := prepareRunOptions(RunOptions{
		ArtifactsDir: artifactsDir,
		Variables:    map[string]string{"OSOK_E2E_SUFFIX": "operator-suffix"},
	}, "scenario", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if prepared.ArtifactsDir != artifactsDir {
		t.Fatalf("artifacts directory = %q, want %q", prepared.ArtifactsDir, artifactsDir)
	}
	if prepared.Variables["OSOK_E2E_SUFFIX"] != "operator-suffix" {
		t.Fatalf("suffix = %q, want operator-suffix", prepared.Variables["OSOK_E2E_SUFFIX"])
	}
}

func TestRenderScenarioDerivesIdentifierSafeSuffix(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "scenario.yaml", "version: 1\nname: identifier\nservice: nosql\ncreate: create.yaml\n")
	writeTestFile(t, root, "create.yaml", `apiVersion: nosql.oracle.com/v1beta1
kind: Table
metadata:
  name: table-${OSOK_E2E_SUFFIX}
spec:
  name: table_${OSOK_E2E_ID}
  compartmentId: ${OCI_COMPARTMENT_ID}
  ddlStatement: CREATE TABLE table_${OSOK_E2E_ID} (id INTEGER, PRIMARY KEY(id))
`)
	scenario, err := LoadScenario(filepath.Join(root, "scenario.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := renderScenario(scenario, t.TempDir(), map[string]string{
		"OCI_COMPARTMENT_ID": "ocid1.compartment.oc1..test",
		"OSOK_E2E_SUFFIX":    "20260831-123456.a",
	})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(rendered.Create)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "table_20260831123456a") {
		t.Fatalf("rendered manifest did not contain safe identifier:\n%s", content)
	}
}

func TestCheckedInLifecycleScenariosRenderAndKeepResourceIdentity(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(filepath.Join("..", "..", "..", "e2e", "scenarios", "*", "*", "scenario.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no checked-in lifecycle scenarios found")
	}
	for _, path := range paths {
		path := path
		t.Run(filepath.ToSlash(path), func(t *testing.T) {
			t.Parallel()
			scenario, err := LoadScenario(path)
			if err != nil {
				t.Fatal(err)
			}
			rendered, err := renderScenario(scenario, t.TempDir(), map[string]string{
				"OCI_AVAILABILITY_DOMAIN":        "example:US-ASHBURN-AD-1",
				"OCI_APIGATEWAY_ID":              "ocid1.apigateway.oc1..scenario",
				"OCI_CLUSTER_ID":                 "ocid1.cluster.oc1..scenario",
				"OCI_COMPUTE_SHAPE":              "VM.Standard.E4.Flex",
				"OCI_COMPARTMENT_ID":             "ocid1.compartment.oc1..scenario",
				"OCI_EMAIL_SENDER_ADDRESS":       "sender@example.com",
				"OCI_FILE_STORAGE_EXPORT_SET_ID": "ocid1.exportset.oc1..scenario",
				"OCI_FILE_SYSTEM_ID":             "ocid1.filesystem.oc1..scenario",
				"OCI_IMAGE_ID":                   "ocid1.image.oc1..scenario",
				"OCI_INSTANCE_POOL_ID":           "ocid1.instancepool.oc1..scenario",
				"OCI_KUBERNETES_VERSION":         "v1.36.1",
				"OCI_LOG_GROUP_ID":               "ocid1.loggroup.oc1..scenario",
				"OCI_MYSQL_ADMIN_PASSWORD":       "ScenarioPass123!",
				"OCI_MYSQL_SHAPE":                "MySQL.2",
				"OCI_NOTIFICATION_ENDPOINT":      "ocid1.fnfunc.oc1..scenario",
				"OCI_NOTIFICATION_PROTOCOL":      "ORACLE_FUNCTIONS",
				"OCI_NOTIFICATION_TOPIC_ID":      "ocid1.onstopic.oc1..scenario",
				"OCI_POD_SUBNET_ID":              "ocid1.subnet.oc1..pods-scenario",
				"OCI_PSQL_ADMIN_PASSWORD":        "ScenarioPass123!",
				"OCI_PSQL_DB_VERSION":            "16",
				"OCI_PSQL_SHAPE":                 "VM.Standard.E5.Flex",
				"OCI_REDIS_SOFTWARE_VERSION":     "VALKEY_7_2",
				"OCI_TENANCY_ID":                 "ocid1.tenancy.oc1..scenario",
				"OCI_REGION":                     "us-ashburn-1",
				"OCI_SUBNET_ID":                  "ocid1.subnet.oc1..scenario",
				"OCI_VCN_ID":                     "ocid1.vcn.oc1..scenario",
				"OSOK_E2E_SUFFIX":                "20260917-123456-0123456789ab",
			})
			if err != nil {
				t.Fatal(err)
			}
			createRef, err := manifestResourceRef(rendered.Create, scenario.Namespace)
			if err != nil {
				t.Fatal(err)
			}
			if rendered.Update != "" {
				updateRef, err := manifestResourceRef(rendered.Update, scenario.Namespace)
				if err != nil {
					t.Fatal(err)
				}
				if createRef != updateRef {
					t.Fatalf("create ref = %#v, update ref = %#v", createRef, updateRef)
				}
			}
		})
	}
}

func TestEvaluateReadinessRequiresEveryConfiguredCategory(t *testing.T) {
	t.Parallel()

	object := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"generation": int64(2)},
		"status": map[string]any{
			"lifecycleState": "UPDATING",
			"status": map[string]any{
				"ocid": "ocid1.test",
				"conditions": []any{map[string]any{
					"type": "Active", "status": "True", "observedGeneration": int64(1),
				}},
			},
		},
	}}
	ready, failed, _ := evaluateReadiness(object, ReadyAssertions{
		ConditionTypes:    []string{"Active"},
		LifecycleStates:   []string{"ACTIVE"},
		RequireOCID:       true,
		ObserveGeneration: true,
		FieldsEqual:       []FieldEquality{{Desired: "spec.displayName", Observed: "status.displayName"}},
	}, []string{"Failed"})
	if ready || failed {
		t.Fatalf("evaluateReadiness() = ready:%t failed:%t", ready, failed)
	}
}

func TestEvaluateReadinessUsesLatestCondition(t *testing.T) {
	t.Parallel()

	object := &unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{
			"status": map[string]any{
				"reason": "Active",
				"conditions": []any{
					map[string]any{"type": "Failed", "status": "True", "message": "transient"},
					map[string]any{"type": "Active", "status": "True", "message": "recovered"},
				},
			},
		},
	}}
	ready, failed, _ := evaluateReadiness(object, ReadyAssertions{ConditionTypes: []string{"Active"}}, []string{"Failed"})
	if !ready || failed {
		t.Fatalf("evaluateReadiness() = ready:%t failed:%t, want recovered latest condition", ready, failed)
	}

	conditions, _, _ := unstructured.NestedSlice(object.Object, "status", "status", "conditions")
	conditions = append(conditions, map[string]any{"type": "Failed", "status": "False", "message": "terminal"})
	if err := unstructured.SetNestedSlice(object.Object, conditions, "status", "status", "conditions"); err != nil {
		t.Fatal(err)
	}
	if err := unstructured.SetNestedField(object.Object, "Failed", "status", "status", "reason"); err != nil {
		t.Fatal(err)
	}
	ready, failed, _ = evaluateReadiness(object, ReadyAssertions{ConditionTypes: []string{"Active"}}, []string{"Failed"})
	if ready || !failed {
		t.Fatalf("evaluateReadiness() = ready:%t failed:%t, want latest failure", ready, failed)
	}
}

func writeTestFile(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func unusedEnvironmentVariable() string {
	for index := 0; ; index++ {
		name := fmt.Sprintf("OSOK_E2E_TEST_REQUIRED_UNSET_%d", index)
		if _, exists := os.LookupEnv(name); !exists {
			return name
		}
	}
}

func testBudgetManifest(displayName string) string {
	return fmt.Sprintf(`apiVersion: budget.oracle.com/v1beta1
kind: Budget
metadata:
  name: budget-demo
spec:
  compartmentId: ${OCI_COMPARTMENT_ID}
  amount: 100
  resetPeriod: MONTHLY
  displayName: %s-${OSOK_E2E_SUFFIX}
`, displayName)
}

func resourceJSON(generation, observed int64, conditionType, conditionStatus, lifecycle, ocid string) string {
	return fmt.Sprintf(`{
  "apiVersion":"budget.oracle.com/v1beta1",
  "kind":"Budget",
  "metadata":{"name":"budget-demo","namespace":"default","generation":%d},
  "status":{
    "lifecycleState":%q,
    "status":{
      "reason":%q,
      "ocid":%q,
      "conditions":[{"type":%q,"status":%q,"observedGeneration":%d,"reason":"","message":""}]
    }
  }
}`, generation, lifecycle, conditionType, ocid, conditionType, conditionStatus, observed)
}
