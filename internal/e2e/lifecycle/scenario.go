/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Package lifecycle executes OSOK resource create, update, and delete scenarios
// against a Kubernetes cluster.
package lifecycle

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const scenarioVersion = 1

// Scenario describes one live OSOK resource lifecycle.
type Scenario struct {
	Version               int                      `yaml:"version"`
	Name                  string                   `yaml:"name"`
	Service               string                   `yaml:"service"`
	Namespace             string                   `yaml:"namespace,omitempty"`
	Timeout               string                   `yaml:"timeout,omitempty"`
	PollInterval          string                   `yaml:"pollInterval,omitempty"`
	Dependencies          []string                 `yaml:"dependencies,omitempty"`
	Create                string                   `yaml:"create"`
	Update                string                   `yaml:"update,omitempty"`
	Ready                 ReadyAssertions          `yaml:"ready,omitempty"`
	FailureConditionTypes []string                 `yaml:"failureConditionTypes,omitempty"`
	RelatedObjects        []RelatedObjectAssertion `yaml:"relatedObjects,omitempty"`
	Delete                DeleteSpec               `yaml:"delete,omitempty"`
	CleanupOnFailure      *bool                    `yaml:"cleanupOnFailure,omitempty"`
}

// ReadyAssertions define observable Kubernetes evidence for a converged CR.
// Every configured category must pass.
type ReadyAssertions struct {
	ConditionTypes    []string        `yaml:"conditionTypes,omitempty"`
	LifecycleStates   []string        `yaml:"lifecycleStates,omitempty"`
	RequireOCID       bool            `yaml:"requireOCID,omitempty"`
	ObserveGeneration bool            `yaml:"observeGeneration,omitempty"`
	FieldsEqual       []FieldEquality `yaml:"fieldsEqual,omitempty"`
}

// FieldEquality waits until the observed object field equals its desired field.
// Dot-separated paths are relative to the custom resource root.
type FieldEquality struct {
	Desired  string `yaml:"desired"`
	Observed string `yaml:"observed"`
}

// RelatedObjectAssertion verifies a Kubernetes side effect owned by the
// primary resource. Empty name or namespace values inherit from that resource.
type RelatedObjectAssertion struct {
	APIVersion         string   `yaml:"apiVersion"`
	Kind               string   `yaml:"kind"`
	Name               string   `yaml:"name,omitempty"`
	Namespace          string   `yaml:"namespace,omitempty"`
	RequiredDataKeys   []string `yaml:"requiredDataKeys,omitempty"`
	DeleteWithResource bool     `yaml:"deleteWithResource,omitempty"`
}

// DeleteSpec configures cleanup of the primary resource and dependencies.
type DeleteSpec struct {
	Enabled             *bool  `yaml:"enabled,omitempty"`
	Timeout             string `yaml:"timeout,omitempty"`
	CleanupDependencies *bool  `yaml:"cleanupDependencies,omitempty"`
}

type loadedScenario struct {
	Scenario
	Root            string
	TimeoutValue    time.Duration
	PollValue       time.Duration
	DeleteTimeout   time.Duration
	CleanupFailure  bool
	DeleteEnabled   bool
	CleanupDeps     bool
	CreatePath      string
	UpdatePath      string
	DependencyPaths []string
}

// LoadScenario validates a scenario and resolves only files contained by its
// directory. Symlinks and traversal cannot escape the scenario root.
func LoadScenario(path string) (*loadedScenario, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read lifecycle scenario: %w", err)
	}
	var scenario Scenario
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&scenario); err != nil {
		return nil, fmt.Errorf("decode lifecycle scenario: %w", err)
	}
	if scenario.Version != scenarioVersion {
		return nil, fmt.Errorf("scenario version %d is unsupported; expected %d", scenario.Version, scenarioVersion)
	}
	if strings.TrimSpace(scenario.Name) == "" {
		return nil, errors.New("scenario name is required")
	}
	if strings.TrimSpace(scenario.Service) == "" {
		return nil, errors.New("scenario service is required")
	}
	if strings.TrimSpace(scenario.Create) == "" {
		return nil, errors.New("scenario create manifest is required")
	}
	if scenario.Namespace == "" {
		scenario.Namespace = "default"
	}
	if len(scenario.Ready.ConditionTypes) == 0 && len(scenario.Ready.LifecycleStates) == 0 {
		scenario.Ready.ConditionTypes = []string{"Active"}
	}
	for index, equality := range scenario.Ready.FieldsEqual {
		if strings.TrimSpace(equality.Desired) == "" || strings.TrimSpace(equality.Observed) == "" {
			return nil, fmt.Errorf("ready.fieldsEqual[%d] requires desired and observed paths", index)
		}
	}
	for index, related := range scenario.RelatedObjects {
		if strings.TrimSpace(related.APIVersion) == "" || strings.TrimSpace(related.Kind) == "" {
			return nil, fmt.Errorf("relatedObjects[%d] requires apiVersion and kind", index)
		}
		if _, err := schema.ParseGroupVersion(related.APIVersion); err != nil {
			return nil, fmt.Errorf("relatedObjects[%d].apiVersion %q is invalid: %w", index, related.APIVersion, err)
		}
		for keyIndex, key := range related.RequiredDataKeys {
			if strings.TrimSpace(key) == "" {
				return nil, fmt.Errorf("relatedObjects[%d].requiredDataKeys[%d] must not be empty", index, keyIndex)
			}
		}
	}
	if len(scenario.FailureConditionTypes) == 0 {
		scenario.FailureConditionTypes = []string{"Failed"}
	}

	timeout, err := parsePositiveDuration(scenario.Timeout, 30*time.Minute, "timeout")
	if err != nil {
		return nil, err
	}
	poll, err := parsePositiveDuration(scenario.PollInterval, 5*time.Second, "pollInterval")
	if err != nil {
		return nil, err
	}
	deleteTimeout, err := parsePositiveDuration(scenario.Delete.Timeout, timeout, "delete.timeout")
	if err != nil {
		return nil, err
	}

	root, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("resolve scenario root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario root symlinks: %w", err)
	}
	loaded := &loadedScenario{
		Scenario:       scenario,
		Root:           root,
		TimeoutValue:   timeout,
		PollValue:      poll,
		DeleteTimeout:  deleteTimeout,
		CleanupFailure: boolValue(scenario.CleanupOnFailure, true),
		DeleteEnabled:  boolValue(scenario.Delete.Enabled, true),
		CleanupDeps:    boolValue(scenario.Delete.CleanupDependencies, true),
	}
	loaded.CreatePath, err = resolveContainedFile(root, scenario.Create)
	if err != nil {
		return nil, fmt.Errorf("resolve create manifest: %w", err)
	}
	if scenario.Update != "" {
		loaded.UpdatePath, err = resolveContainedFile(root, scenario.Update)
		if err != nil {
			return nil, fmt.Errorf("resolve update manifest: %w", err)
		}
	}
	for _, dependency := range scenario.Dependencies {
		resolved, resolveErr := resolveContainedFile(root, dependency)
		if resolveErr != nil {
			return nil, fmt.Errorf("resolve dependency manifest %q: %w", dependency, resolveErr)
		}
		loaded.DependencyPaths = append(loaded.DependencyPaths, resolved)
	}
	return loaded, nil
}

func resolveContainedFile(root, name string) (string, error) {
	if filepath.IsAbs(name) {
		return "", errors.New("absolute paths are not allowed")
	}
	candidate := filepath.Join(root, filepath.Clean(name))
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes the scenario directory")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("scenario path is not a regular file")
	}
	return resolved, nil
}

func parsePositiveDuration(value string, fallback time.Duration, field string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", field, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be positive", field)
	}
	return duration, nil
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}
