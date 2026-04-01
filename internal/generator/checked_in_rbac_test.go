/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"path/filepath"
	"slices"
	"sort"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/yaml"
)

func TestCheckedInDatabaseAutonomousDatabasePackageRBACMatchesReadOnlySecretSemantics(t *testing.T) {
	controllerPath := filepath.Join(repoRoot(t), "controllers", "database", "autonomousdatabase_controller.go")
	assertFileContains(t, controllerPath, []string{
		`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch`,
		`// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch`,
	})
	assertFileDoesNotContain(t, controllerPath, []string{
		`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`,
		`// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
	})

	assertCoreResourceVerbs(
		t,
		filepath.Join(repoRoot(t), "packages", "database", "install", "generated", "rbac", "role.yaml"),
		map[string][]string{
			"events":  {"create", "patch"},
			"secrets": {"get", "list", "watch"},
		},
	)
}

func assertCoreResourceVerbs(t *testing.T, path string, want map[string][]string) {
	t.Helper()

	role := loadClusterRole(t, path)
	got := collectCoreResourceVerbs(role.Rules, want)

	for resource, wantVerbs := range want {
		gotVerbs := sortedKeys(got[resource])
		if !slices.Equal(gotVerbs, wantVerbs) {
			t.Fatalf("%s %s verbs = %v, want %v", path, resource, gotVerbs, wantVerbs)
		}
	}
}

func loadClusterRole(t *testing.T, path string) rbacv1.ClusterRole {
	t.Helper()

	var role rbacv1.ClusterRole
	if err := yaml.Unmarshal([]byte(readFile(t, path)), &role); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return role
}

func collectCoreResourceVerbs(rules []rbacv1.PolicyRule, tracked map[string][]string) map[string]map[string]struct{} {
	got := make(map[string]map[string]struct{}, len(tracked))
	for _, rule := range rules {
		if !containsString(rule.APIGroups, "") {
			continue
		}
		mergeTrackedResourceVerbs(got, tracked, rule.Resources, rule.Verbs)
	}
	return got
}

func mergeTrackedResourceVerbs(got map[string]map[string]struct{}, tracked map[string][]string, resources, verbs []string) {
	for _, resource := range resources {
		if _, ok := tracked[resource]; !ok {
			continue
		}
		if got[resource] == nil {
			got[resource] = make(map[string]struct{}, len(verbs))
		}
		for _, verb := range verbs {
			got[resource][verb] = struct{}{}
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
