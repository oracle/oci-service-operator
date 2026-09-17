/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package lifecycle

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
)

// RunOptions supplies cluster access, manifest variables, and evidence paths.
type RunOptions struct {
	ScenarioPath    string
	ArtifactsDir    string
	KubectlBinary   string
	Kubeconfig      string
	Variables       map[string]string
	ExpectedService string
	CommandRunner   commandRunner
	Now             func() time.Time
}

// Result is stable machine-readable evidence for one lifecycle run.
type Result struct {
	Scenario     string        `json:"scenario"`
	Service      string        `json:"service"`
	Status       string        `json:"status"`
	StartedAt    time.Time     `json:"started_at"`
	FinishedAt   time.Time     `json:"finished_at"`
	Resource     resourceRef   `json:"resource"`
	Phases       []PhaseResult `json:"phases"`
	Error        string        `json:"error,omitempty"`
	CleanupError string        `json:"cleanup_error,omitempty"`
	RenderedDir  string        `json:"rendered_dir"`
}

// PhaseResult records one deterministic lifecycle phase.
type PhaseResult struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Detail     string    `json:"detail,omitempty"`
}

type runner struct {
	scenario *loadedScenario
	client   *kubectlClient
	options  RunOptions
	result   Result
	now      func() time.Time
	created  bool
	depsDone int
	paths    renderedPaths
}

type renderedPaths struct {
	Create       string
	Update       string
	Dependencies []string
}

// Run executes one scenario and always writes result.json when an artifacts
// directory can be prepared.
func Run(ctx context.Context, options RunOptions) (Result, error) {
	scenario, err := LoadScenario(options.ScenarioPath)
	if err != nil {
		return Result{}, err
	}
	if options.ExpectedService != "" && options.ExpectedService != scenario.Service {
		return Result{}, fmt.Errorf("scenario service %q does not match installed service %q", scenario.Service, options.ExpectedService)
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	options, err = prepareRunOptions(options, scenario.Name, os.TempDir())
	if err != nil {
		return Result{}, err
	}

	r := &runner{
		scenario: scenario,
		client:   newKubectlClient(options.KubectlBinary, options.Kubeconfig, options.CommandRunner),
		options:  options,
		now:      options.Now,
		result: Result{
			Scenario:    scenario.Name,
			Service:     scenario.Service,
			Status:      "running",
			StartedAt:   options.Now().UTC(),
			RenderedDir: filepath.Join(options.ArtifactsDir, "rendered"),
		},
	}
	r.paths, err = renderScenario(scenario, r.result.RenderedDir, options.Variables)
	if err != nil {
		r.finish("failed", err)
		_ = r.writeResult()
		return r.result, err
	}

	runErr := r.execute(ctx)
	if runErr != nil && scenario.CleanupFailure {
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), scenario.DeleteTimeout+time.Minute)
		defer cancelCleanup()
		if cleanupErr := r.cleanup(cleanupCtx); cleanupErr != nil {
			r.result.CleanupError = cleanupErr.Error()
		}
	}
	if runErr == nil {
		r.finish("passed", nil)
	} else {
		r.finish("failed", runErr)
	}
	if writeErr := r.writeResult(); writeErr != nil {
		if runErr != nil {
			return r.result, errors.Join(runErr, writeErr)
		}
		return r.result, writeErr
	}
	return r.result, runErr
}

func (r *runner) execute(ctx context.Context) error {
	for index, dependency := range r.paths.Dependencies {
		if err := r.phase(ctx, fmt.Sprintf("apply_dependency_%d", index+1), func(phaseCtx context.Context) (string, error) {
			output, err := r.client.apply(phaseCtx, dependency, r.scenario.Namespace)
			if err == nil {
				r.depsDone = index + 1
			}
			return strings.TrimSpace(string(output)), err
		}); err != nil {
			return err
		}
	}

	createRef, err := manifestResourceRef(r.paths.Create, r.scenario.Namespace)
	if err != nil {
		return err
	}
	r.result.Resource = createRef
	if r.paths.Update != "" {
		updateRef, refErr := manifestResourceRef(r.paths.Update, r.scenario.Namespace)
		if refErr != nil {
			return refErr
		}
		if updateRef != createRef {
			return fmt.Errorf("update manifest identifies %s/%s, want create resource %s/%s", updateRef.Kind, updateRef.Name, createRef.Kind, createRef.Name)
		}
	}

	if err := r.phase(ctx, "create", func(phaseCtx context.Context) (string, error) {
		output, applyErr := r.client.apply(phaseCtx, r.paths.Create, createRef.Namespace)
		if applyErr == nil {
			r.created = true
		}
		return strings.TrimSpace(string(output)), applyErr
	}); err != nil {
		return err
	}
	if err := r.phase(ctx, "wait_after_create", func(phaseCtx context.Context) (string, error) {
		return r.waitReady(phaseCtx, createRef, r.scenario.TimeoutValue)
	}); err != nil {
		return err
	}
	if len(r.scenario.RelatedObjects) > 0 {
		if err := r.phase(ctx, "verify_related_after_create", func(phaseCtx context.Context) (string, error) {
			return r.waitRelatedReady(phaseCtx, createRef, r.scenario.TimeoutValue)
		}); err != nil {
			return err
		}
	}

	if r.paths.Update != "" {
		if err := r.phase(ctx, "update", func(phaseCtx context.Context) (string, error) {
			output, applyErr := r.client.apply(phaseCtx, r.paths.Update, createRef.Namespace)
			return strings.TrimSpace(string(output)), applyErr
		}); err != nil {
			return err
		}
		if err := r.phase(ctx, "wait_after_update", func(phaseCtx context.Context) (string, error) {
			return r.waitReady(phaseCtx, createRef, r.scenario.TimeoutValue)
		}); err != nil {
			return err
		}
		if len(r.scenario.RelatedObjects) > 0 {
			if err := r.phase(ctx, "verify_related_after_update", func(phaseCtx context.Context) (string, error) {
				return r.waitRelatedReady(phaseCtx, createRef, r.scenario.TimeoutValue)
			}); err != nil {
				return err
			}
		}
	}

	if r.scenario.DeleteEnabled {
		if err := r.phase(ctx, "delete", func(phaseCtx context.Context) (string, error) {
			output, deleteErr := r.client.deleteResource(phaseCtx, createRef)
			return strings.TrimSpace(string(output)), deleteErr
		}); err != nil {
			return err
		}
		if err := r.phase(ctx, "wait_for_deletion", func(phaseCtx context.Context) (string, error) {
			return r.waitDeleted(phaseCtx, createRef, r.scenario.DeleteTimeout)
		}); err != nil {
			return err
		}
		r.created = false
		if hasRelatedDeletionAssertions(r.scenario.RelatedObjects) {
			if err := r.phase(ctx, "verify_related_deletion", func(phaseCtx context.Context) (string, error) {
				return r.waitRelatedDeleted(phaseCtx, createRef, r.scenario.DeleteTimeout)
			}); err != nil {
				return err
			}
		}
	}
	if r.scenario.CleanupDeps && r.depsDone > 0 {
		dependencyCount := r.depsDone
		if err := r.phase(ctx, "cleanup_dependencies", func(phaseCtx context.Context) (string, error) {
			err := r.cleanupDependencies(phaseCtx)
			return fmt.Sprintf("deleted %d dependency manifest(s)", dependencyCount), err
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *runner) phase(ctx context.Context, name string, run func(context.Context) (string, error)) error {
	phase := PhaseResult{Name: name, Status: "running", StartedAt: r.now().UTC()}
	detail, err := run(ctx)
	phase.FinishedAt = r.now().UTC()
	phase.Detail = detail
	if err != nil {
		phase.Status = "failed"
		r.result.Phases = append(r.result.Phases, phase)
		return fmt.Errorf("phase %s: %w", name, err)
	}
	phase.Status = "passed"
	r.result.Phases = append(r.result.Phases, phase)
	return nil
}

func (r *runner) waitReady(ctx context.Context, ref resourceRef, timeout time.Duration) (string, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	deadline := r.now().Add(timeout)
	var last string
	for {
		object, err := r.client.get(waitCtx, ref)
		if err == nil {
			ready, failed, detail := evaluateReadiness(object, r.scenario.Ready, r.scenario.FailureConditionTypes)
			last = detail
			if failed {
				return detail, errors.New(detail)
			}
			if ready {
				return detail, nil
			}
		} else if !errors.Is(err, errResourceNotFound) {
			last = err.Error()
		}
		if !r.now().Before(deadline) {
			return last, fmt.Errorf("timed out after %s waiting for %s/%s to become ready: %s", timeout, ref.Kind, ref.Name, last)
		}
		if err := sleepContext(waitCtx, r.scenario.PollValue); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return last, fmt.Errorf("timed out after %s waiting for %s/%s to become ready: %s", timeout, ref.Kind, ref.Name, last)
			}
			return last, err
		}
	}
}

func (r *runner) waitDeleted(ctx context.Context, ref resourceRef, timeout time.Duration) (string, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	deadline := r.now().Add(timeout)
	for {
		_, err := r.client.get(waitCtx, ref)
		if errors.Is(err, errResourceNotFound) {
			return "resource no longer exists", nil
		}
		if err != nil {
			return err.Error(), err
		}
		if !r.now().Before(deadline) {
			return "resource still exists", fmt.Errorf("timed out after %s waiting for %s/%s deletion", timeout, ref.Kind, ref.Name)
		}
		if err := sleepContext(waitCtx, r.scenario.PollValue); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return "resource still exists", fmt.Errorf("timed out after %s waiting for %s/%s deletion", timeout, ref.Kind, ref.Name)
			}
			return "", err
		}
	}
}

func (r *runner) waitRelatedReady(ctx context.Context, primary resourceRef, timeout time.Duration) (string, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	deadline := r.now().Add(timeout)
	var last string
	for {
		ready := true
		var verified []string
		for _, assertion := range r.scenario.RelatedObjects {
			ref, err := relatedObjectRef(primary, assertion)
			if err != nil {
				return last, err
			}
			object, err := r.client.get(waitCtx, ref)
			if err != nil {
				ready = false
				last = fmt.Sprintf("waiting for %s/%s: %v", ref.Kind, ref.Name, err)
				continue
			}
			data, _, _ := unstructured.NestedStringMap(object.Object, "data")
			for _, key := range assertion.RequiredDataKeys {
				if strings.TrimSpace(data[key]) == "" {
					ready = false
					last = fmt.Sprintf("waiting for %s/%s data key %q", ref.Kind, ref.Name, key)
				}
			}
			if ready {
				verified = append(verified, fmt.Sprintf("%s/%s keys=%s", ref.Kind, ref.Name, strings.Join(assertion.RequiredDataKeys, ",")))
			}
		}
		if ready {
			return strings.Join(verified, "; "), nil
		}
		if !r.now().Before(deadline) {
			return last, fmt.Errorf("timed out after %s waiting for related objects: %s", timeout, last)
		}
		if err := sleepContext(waitCtx, r.scenario.PollValue); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return last, fmt.Errorf("timed out after %s waiting for related objects: %s", timeout, last)
			}
			return last, err
		}
	}
}

func (r *runner) waitRelatedDeleted(ctx context.Context, primary resourceRef, timeout time.Duration) (string, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	deadline := r.now().Add(timeout)
	var last string
	for {
		deleted := true
		var verified []string
		for _, assertion := range r.scenario.RelatedObjects {
			if !assertion.DeleteWithResource {
				continue
			}
			ref, err := relatedObjectRef(primary, assertion)
			if err != nil {
				return last, err
			}
			_, err = r.client.get(waitCtx, ref)
			if errors.Is(err, errResourceNotFound) {
				verified = append(verified, fmt.Sprintf("%s/%s deleted", ref.Kind, ref.Name))
				continue
			}
			deleted = false
			if err != nil {
				last = fmt.Sprintf("waiting for %s/%s deletion: %v", ref.Kind, ref.Name, err)
			} else {
				last = fmt.Sprintf("waiting for %s/%s deletion", ref.Kind, ref.Name)
			}
		}
		if deleted {
			return strings.Join(verified, "; "), nil
		}
		if !r.now().Before(deadline) {
			return last, fmt.Errorf("timed out after %s waiting for related object deletion: %s", timeout, last)
		}
		if err := sleepContext(waitCtx, r.scenario.PollValue); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return last, fmt.Errorf("timed out after %s waiting for related object deletion: %s", timeout, last)
			}
			return last, err
		}
	}
}

func relatedObjectRef(primary resourceRef, assertion RelatedObjectAssertion) (resourceRef, error) {
	groupVersion, err := schema.ParseGroupVersion(assertion.APIVersion)
	if err != nil {
		return resourceRef{}, fmt.Errorf("parse related object apiVersion %q: %w", assertion.APIVersion, err)
	}
	name := strings.TrimSpace(assertion.Name)
	if name == "" {
		name = primary.Name
	}
	namespace := strings.TrimSpace(assertion.Namespace)
	if namespace == "" {
		namespace = primary.Namespace
	}
	return resourceRef{
		Group:     groupVersion.Group,
		Version:   groupVersion.Version,
		Kind:      assertion.Kind,
		Name:      name,
		Namespace: namespace,
	}, nil
}

func hasRelatedDeletionAssertions(assertions []RelatedObjectAssertion) bool {
	for _, assertion := range assertions {
		if assertion.DeleteWithResource {
			return true
		}
	}
	return false
}

func (r *runner) cleanup(ctx context.Context) error {
	var cleanupErrors []error
	if r.created && r.result.Resource.Name != "" {
		if _, err := r.client.deleteResource(ctx, r.result.Resource); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		} else if _, err := r.waitDeleted(ctx, r.result.Resource, r.scenario.DeleteTimeout); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	if r.scenario.CleanupDeps {
		if err := r.cleanupDependencies(ctx); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return errors.Join(cleanupErrors...)
}

func (r *runner) cleanupDependencies(ctx context.Context) error {
	var cleanupErrors []error
	for index := r.depsDone - 1; index >= 0; index-- {
		if _, err := r.client.deleteFile(ctx, r.paths.Dependencies[index], r.scenario.Namespace, r.scenario.DeleteTimeout); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	r.depsDone = 0
	return errors.Join(cleanupErrors...)
}

func (r *runner) finish(status string, runErr error) {
	r.result.Status = status
	r.result.FinishedAt = r.now().UTC()
	if runErr != nil {
		r.result.Error = runErr.Error()
	}
}

func (r *runner) writeResult() error {
	content, err := json.MarshalIndent(r.result, "", "  ")
	if err != nil {
		return fmt.Errorf("encode lifecycle result: %w", err)
	}
	content = append(content, '\n')
	path := filepath.Join(r.options.ArtifactsDir, "result.json")
	temporary, err := os.CreateTemp(r.options.ArtifactsDir, ".result.json.tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary lifecycle result: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return fmt.Errorf("write lifecycle result: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close lifecycle result: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish lifecycle result: %w", err)
	}
	return nil
}

func prepareRunOptions(options RunOptions, scenarioName, temporaryRoot string) (RunOptions, error) {
	if options.Now == nil {
		options.Now = time.Now
	}

	effectiveSuffix := os.Getenv("OSOK_E2E_SUFFIX")
	if value, ok := options.Variables["OSOK_E2E_SUFFIX"]; ok {
		effectiveSuffix = value
	}
	needsGeneratedIdentity := options.ArtifactsDir == "" || effectiveSuffix == ""
	generatedSuffix := ""
	if needsGeneratedIdentity {
		var err error
		generatedSuffix, err = newRunSuffix(options.Now())
		if err != nil {
			return RunOptions{}, err
		}
	}
	if effectiveSuffix == "" {
		variables := make(map[string]string, len(options.Variables)+1)
		for key, value := range options.Variables {
			variables[key] = value
		}
		variables["OSOK_E2E_SUFFIX"] = generatedSuffix
		options.Variables = variables
	}

	if options.ArtifactsDir == "" {
		artifactsDir, err := createDefaultArtifactsDir(temporaryRoot, scenarioName, generatedSuffix)
		if err != nil {
			return RunOptions{}, err
		}
		options.ArtifactsDir = artifactsDir
		return options, nil
	}
	if err := os.MkdirAll(options.ArtifactsDir, 0o755); err != nil {
		return RunOptions{}, fmt.Errorf("create lifecycle artifact directory: %w", err)
	}
	return options, nil
}

func newRunSuffix(now time.Time) (string, error) {
	entropy := make([]byte, 6)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("generate lifecycle run suffix: %w", err)
	}
	return now.UTC().Format("20060102-150405") + "-" + hex.EncodeToString(entropy), nil
}

func createDefaultArtifactsDir(temporaryRoot, scenarioName, runSuffix string) (string, error) {
	root := filepath.Join(temporaryRoot, "osok-e2e", scenarioName)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("create lifecycle artifact root: %w", err)
	}
	artifactsDir, err := os.MkdirTemp(root, runSuffix+"-")
	if err != nil {
		return "", fmt.Errorf("create lifecycle artifact directory: %w", err)
	}
	return artifactsDir, nil
}

func renderScenario(scenario *loadedScenario, destination string, variables map[string]string) (renderedPaths, error) {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return renderedPaths{}, fmt.Errorf("create rendered manifest directory: %w", err)
	}
	values := map[string]string{}
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if found {
			values[key] = value
		}
	}
	for key, value := range variables {
		values[key] = value
	}
	if values["OSOK_E2E_SUFFIX"] == "" {
		suffix, err := newRunSuffix(time.Now())
		if err != nil {
			return renderedPaths{}, err
		}
		values["OSOK_E2E_SUFFIX"] = suffix
	}
	if values["OSOK_E2E_ID"] == "" {
		var identifier strings.Builder
		for _, character := range values["OSOK_E2E_SUFFIX"] {
			if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' {
				identifier.WriteRune(character)
			}
		}
		if identifier.Len() == 0 {
			identifier.WriteString("e2e")
		}
		values["OSOK_E2E_ID"] = identifier.String()
	}
	values["OSOK_E2E_NAMESPACE"] = scenario.Namespace

	render := func(source, name string) (string, error) {
		if source == "" {
			return "", nil
		}
		content, err := os.ReadFile(source)
		if err != nil {
			return "", err
		}
		missing := map[string]struct{}{}
		rendered := os.Expand(string(content), func(key string) string {
			if value, ok := values[key]; ok {
				return value
			}
			missing[key] = struct{}{}
			return "${" + key + "}"
		})
		if len(missing) > 0 {
			keys := make([]string, 0, len(missing))
			for key := range missing {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			return "", fmt.Errorf("manifest %s requires unset variables: %s", source, strings.Join(keys, ", "))
		}
		target := filepath.Join(destination, name)
		if err := os.WriteFile(target, []byte(rendered), 0o600); err != nil {
			return "", err
		}
		return target, nil
	}

	var paths renderedPaths
	var err error
	paths.Create, err = render(scenario.CreatePath, "create.yaml")
	if err != nil {
		return paths, err
	}
	paths.Update, err = render(scenario.UpdatePath, "update.yaml")
	if err != nil {
		return paths, err
	}
	for index, dependency := range scenario.DependencyPaths {
		target, renderErr := render(dependency, fmt.Sprintf("dependency-%02d.yaml", index+1))
		if renderErr != nil {
			return paths, renderErr
		}
		paths.Dependencies = append(paths.Dependencies, target)
	}
	return paths, nil
}

func manifestResourceRef(path, fallbackNamespace string) (resourceRef, error) {
	content, err := os.Open(path)
	if err != nil {
		return resourceRef{}, err
	}
	defer content.Close()
	decoder := utilyaml.NewYAMLOrJSONDecoder(bufio.NewReader(content), 4096)
	var raw map[string]any
	if err := decoder.Decode(&raw); err != nil {
		return resourceRef{}, fmt.Errorf("decode resource manifest %s: %w", path, err)
	}
	if len(raw) == 0 {
		return resourceRef{}, fmt.Errorf("resource manifest %s is empty", path)
	}
	var extra map[string]any
	if err := decoder.Decode(&extra); err != nil && !errors.Is(err, io.EOF) {
		return resourceRef{}, fmt.Errorf("check resource manifest %s documents: %w", path, err)
	}
	if len(extra) > 0 {
		return resourceRef{}, fmt.Errorf("resource manifest %s must contain exactly one document", path)
	}
	object := &unstructured.Unstructured{Object: raw}
	gv, err := schema.ParseGroupVersion(object.GetAPIVersion())
	if err != nil {
		return resourceRef{}, fmt.Errorf("parse apiVersion in %s: %w", path, err)
	}
	if object.GetKind() == "" || object.GetName() == "" {
		return resourceRef{}, fmt.Errorf("resource manifest %s requires kind and metadata.name", path)
	}
	namespace := object.GetNamespace()
	if namespace == "" {
		namespace = fallbackNamespace
	}
	return resourceRef{
		Group:     gv.Group,
		Version:   gv.Version,
		Kind:      object.GetKind(),
		Name:      object.GetName(),
		Namespace: namespace,
	}, nil
}

func evaluateReadiness(object *unstructured.Unstructured, assertions ReadyAssertions, failureTypes []string) (bool, bool, string) {
	conditions, _, _ := unstructured.NestedSlice(object.Object, "status", "status", "conditions")
	var messages []string
	var observedGeneration int64
	var lastConditionType string
	for _, raw := range conditions {
		condition, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		conditionType, _, _ := unstructured.NestedString(condition, "type")
		status, _, _ := unstructured.NestedString(condition, "status")
		if conditionType != "" {
			lastConditionType = conditionType
		}
		generation, _, _ := unstructured.NestedInt64(condition, "observedGeneration")
		if generation > observedGeneration {
			observedGeneration = generation
		}
		reason, _, _ := unstructured.NestedString(condition, "reason")
		message, _, _ := unstructured.NestedString(condition, "message")
		if conditionType != "" {
			messages = append(messages, fmt.Sprintf("%s=%s reason=%s message=%s", conditionType, status, reason, message))
		}
	}
	currentConditionType, _, _ := unstructured.NestedString(object.Object, "status", "status", "reason")
	if strings.TrimSpace(currentConditionType) == "" {
		currentConditionType = lastConditionType
	}
	for _, failureType := range failureTypes {
		if strings.EqualFold(currentConditionType, failureType) {
			return false, true, strings.Join(messages, "; ")
		}
	}

	conditionsReady := len(assertions.ConditionTypes) == 0
	for _, readyType := range assertions.ConditionTypes {
		if strings.EqualFold(currentConditionType, readyType) {
			conditionsReady = true
			break
		}
	}
	lifecycle, _, _ := unstructured.NestedString(object.Object, "status", "lifecycleState")
	lifecycleReady := len(assertions.LifecycleStates) == 0
	for _, readyState := range assertions.LifecycleStates {
		if strings.EqualFold(lifecycle, readyState) {
			lifecycleReady = true
			break
		}
	}
	ocid, _, _ := unstructured.NestedString(object.Object, "status", "status", "ocid")
	if ocid == "" {
		ocid, _, _ = unstructured.NestedString(object.Object, "status", "id")
	}
	ocidReady := !assertions.RequireOCID || strings.TrimSpace(ocid) != ""
	generationReady := true
	if assertions.ObserveGeneration {
		generationReady = observedGeneration >= object.GetGeneration()
	}
	fieldsReady := true
	var fieldDetails []string
	for _, equality := range assertions.FieldsEqual {
		desired, desiredFound, desiredErr := unstructured.NestedFieldNoCopy(object.Object, strings.Split(equality.Desired, ".")...)
		observed, observedFound, observedErr := unstructured.NestedFieldNoCopy(object.Object, strings.Split(equality.Observed, ".")...)
		equal := desiredErr == nil && observedErr == nil && desiredFound && observedFound && reflect.DeepEqual(desired, observed)
		if !equal {
			fieldsReady = false
		}
		fieldDetails = append(fieldDetails, fmt.Sprintf("%s=%v %s=%v equal=%t", equality.Desired, desired, equality.Observed, observed, equal))
	}
	detail := fmt.Sprintf("reason=%s conditions=[%s] lifecycle=%s ocid=%t generation=%d observedGeneration=%d",
		currentConditionType, strings.Join(messages, "; "), lifecycle, ocid != "", object.GetGeneration(), observedGeneration)
	if len(fieldDetails) > 0 {
		detail += " fields=[" + strings.Join(fieldDetails, "; ") + "]"
	}
	return conditionsReady && lifecycleReady && ocidReady && generationReady && fieldsReady, false, detail
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
