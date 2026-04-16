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

func TestCheckedInPSQLDbSystemPackageRBACMatchesSecretAndEventRecorderSemantics(t *testing.T) {
	controllerPath := filepath.Join(repoRoot(t), "controllers", "psql", "dbsystem_controller.go")
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
		filepath.Join(repoRoot(t), "packages", "psql", "install", "generated", "rbac", "role.yaml"),
		map[string][]string{
			"events":  {"create", "patch"},
			"secrets": {"get", "list", "watch"},
		},
	)
}

func TestCheckedInRedisClusterPackageRBACMatchesEventRecorderSemantics(t *testing.T) {
	controllerPath := filepath.Join(repoRoot(t), "controllers", "redis", "rediscluster_controller.go")
	assertFileContains(t, controllerPath, []string{
		`// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch`,
	})
	assertFileDoesNotContain(t, controllerPath, []string{
		`// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
	})

	assertCoreResourceVerbs(
		t,
		filepath.Join(repoRoot(t), "packages", "redis", "install", "generated", "rbac", "role.yaml"),
		map[string][]string{
			"events": {"create", "patch"},
		},
	)
}

func TestCheckedInNoSQLPackageRBACMatchesEventRecorderSemantics(t *testing.T) {
	controllerPaths, err := filepath.Glob(filepath.Join(repoRoot(t), "controllers", "nosql", "*_controller.go"))
	if err != nil {
		t.Fatalf("Glob(nosql controllers) error = %v", err)
	}
	if len(controllerPaths) == 0 {
		t.Fatal("nosql controllers were not found")
	}
	sort.Strings(controllerPaths)

	for _, controllerPath := range controllerPaths {
		assertFileContains(t, controllerPath, []string{
			`// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch`,
		})
		assertFileDoesNotContain(t, controllerPath, []string{
			`// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
		})
	}

	assertCoreResourceVerbs(
		t,
		filepath.Join(repoRoot(t), "packages", "nosql", "install", "generated", "rbac", "role.yaml"),
		map[string][]string{
			"events": {"create", "patch"},
		},
	)
}

func TestCheckedInFunctionsPackageRBACMatchesSecretAndEventRecorderSemantics(t *testing.T) {
	assertFileContains(t, filepath.Join(repoRoot(t), "controllers", "functions", "application_controller.go"), []string{
		`// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch`,
	})
	assertFileContains(t, filepath.Join(repoRoot(t), "controllers", "functions", "function_controller.go"), []string{
		`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`,
		`// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch`,
	})
	assertFileDoesNotContain(t, filepath.Join(repoRoot(t), "controllers", "functions", "application_controller.go"), []string{
		`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`,
	})

	assertCoreResourceVerbs(
		t,
		filepath.Join(repoRoot(t), "packages", "functions", "install", "generated", "rbac", "role.yaml"),
		map[string][]string{
			"events":  {"create", "patch"},
			"secrets": {"create", "delete", "get", "list", "patch", "update", "watch"},
		},
	)
}

func TestCheckedInGeneratedControllerPackagesGrantDefaultEventRecorderSemantics(t *testing.T) {
	tests := []struct {
		name            string
		controllerPaths []string
		rolePath        string
		wantCoreVerbs   map[string][]string
	}{
		{
			name:            "analytics",
			controllerPaths: []string{filepath.Join(repoRoot(t), "controllers", "analytics", "analyticsinstance_controller.go")},
			rolePath:        filepath.Join(repoRoot(t), "packages", "analytics", "install", "generated", "rbac", "role.yaml"),
			wantCoreVerbs: map[string][]string{
				"events": {"create", "patch"},
			},
		},
		{
			name: "containerengine",
			controllerPaths: []string{
				filepath.Join(repoRoot(t), "controllers", "containerengine", "cluster_controller.go"),
				filepath.Join(repoRoot(t), "controllers", "containerengine", "nodepool_controller.go"),
			},
			rolePath: filepath.Join(repoRoot(t), "packages", "containerengine", "install", "generated", "rbac", "role.yaml"),
			wantCoreVerbs: map[string][]string{
				"events": {"create", "patch"},
			},
		},
		{
			name:            "containerinstances",
			controllerPaths: []string{filepath.Join(repoRoot(t), "controllers", "containerinstances", "containerinstance_controller.go")},
			rolePath:        filepath.Join(repoRoot(t), "packages", "containerinstances", "install", "generated", "rbac", "role.yaml"),
			wantCoreVerbs: map[string][]string{
				"events": {"create", "patch"},
			},
		},
		{
			name:            "core",
			controllerPaths: []string{filepath.Join(repoRoot(t), "controllers", "core", "instance_controller.go")},
			rolePath:        filepath.Join(repoRoot(t), "packages", "core", "install", "generated", "rbac", "role.yaml"),
			wantCoreVerbs: map[string][]string{
				"events": {"create", "patch"},
			},
		},
		{
			name: "core-network",
			controllerPaths: []string{
				filepath.Join(repoRoot(t), "controllers", "core", "drg_controller.go"),
				filepath.Join(repoRoot(t), "controllers", "core", "internetgateway_controller.go"),
				filepath.Join(repoRoot(t), "controllers", "core", "natgateway_controller.go"),
				filepath.Join(repoRoot(t), "controllers", "core", "networksecuritygroup_controller.go"),
				filepath.Join(repoRoot(t), "controllers", "core", "routetable_controller.go"),
				filepath.Join(repoRoot(t), "controllers", "core", "securitylist_controller.go"),
				filepath.Join(repoRoot(t), "controllers", "core", "servicegateway_controller.go"),
				filepath.Join(repoRoot(t), "controllers", "core", "subnet_controller.go"),
				filepath.Join(repoRoot(t), "controllers", "core", "vcn_controller.go"),
			},
			rolePath: filepath.Join(repoRoot(t), "packages", "core-network", "install", "generated", "rbac", "role.yaml"),
			wantCoreVerbs: map[string][]string{
				"events": {"create", "patch"},
			},
		},
		{
			name:            "keymanagement",
			controllerPaths: []string{filepath.Join(repoRoot(t), "controllers", "keymanagement", "vault_controller.go")},
			rolePath:        filepath.Join(repoRoot(t), "packages", "keymanagement", "install", "generated", "rbac", "role.yaml"),
			wantCoreVerbs: map[string][]string{
				"events": {"create", "patch"},
			},
		},
		{
			name:            "mysql",
			controllerPaths: []string{filepath.Join(repoRoot(t), "controllers", "mysql", "dbsystem_controller.go")},
			rolePath:        filepath.Join(repoRoot(t), "packages", "mysql", "install", "generated", "rbac", "role.yaml"),
			wantCoreVerbs: map[string][]string{
				"events":  {"create", "patch"},
				"secrets": {"create", "delete", "get", "list", "update", "watch"},
			},
		},
		{
			name:            "objectstorage",
			controllerPaths: []string{filepath.Join(repoRoot(t), "controllers", "objectstorage", "bucket_controller.go")},
			rolePath:        filepath.Join(repoRoot(t), "packages", "objectstorage", "install", "generated", "rbac", "role.yaml"),
			wantCoreVerbs: map[string][]string{
				"events": {"create", "patch"},
			},
		},
		{
			name:            "opensearch",
			controllerPaths: []string{filepath.Join(repoRoot(t), "controllers", "opensearch", "opensearchcluster_controller.go")},
			rolePath:        filepath.Join(repoRoot(t), "packages", "opensearch", "install", "generated", "rbac", "role.yaml"),
			wantCoreVerbs: map[string][]string{
				"events": {"create", "patch"},
			},
		},
		{
			name:            "queue",
			controllerPaths: []string{filepath.Join(repoRoot(t), "controllers", "queue", "queue_controller.go")},
			rolePath:        filepath.Join(repoRoot(t), "packages", "queue", "install", "generated", "rbac", "role.yaml"),
			wantCoreVerbs: map[string][]string{
				"events":  {"create", "patch"},
				"secrets": {"create", "delete", "get", "list", "update", "watch"},
			},
		},
		{
			name:            "streaming",
			controllerPaths: []string{filepath.Join(repoRoot(t), "controllers", "streaming", "stream_controller.go")},
			rolePath:        filepath.Join(repoRoot(t), "packages", "streaming", "install", "generated", "rbac", "role.yaml"),
			wantCoreVerbs: map[string][]string{
				"events":  {"create", "patch"},
				"secrets": {"create", "delete", "get", "list", "update", "watch"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, controllerPath := range test.controllerPaths {
				assertControllerHasEventRecorderRBAC(t, controllerPath)
			}
			assertCoreResourceVerbs(t, test.rolePath, test.wantCoreVerbs)
		})
	}
}

func TestCheckedInIdentityPackageRBACUsesActualResourceNames(t *testing.T) {
	assertFileContains(t, filepath.Join(repoRoot(t), "controllers", "identity", "compartment_controller.go"), []string{
		"// +kubebuilder:rbac:groups=identity.oracle.com,resources=compartments,verbs=get;list;watch;create;update;patch;delete",
		"// +kubebuilder:rbac:groups=identity.oracle.com,resources=compartments/status,verbs=get;update;patch",
		"// +kubebuilder:rbac:groups=identity.oracle.com,resources=compartments/finalizers,verbs=update",
		`// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch`,
	})
	assertFileDoesNotContain(t, filepath.Join(repoRoot(t), "controllers", "identity", "compartment_controller.go"), []string{
		"// +kubebuilder:rbac:groups=identity.oracle.com,resources=compartmentes,verbs=get;list;watch;create;update;patch;delete",
	})

	assertCoreResourceVerbs(
		t,
		filepath.Join(repoRoot(t), "packages", "identity", "install", "generated", "rbac", "role.yaml"),
		map[string][]string{
			"events": {"create", "patch"},
		},
	)

	assertAPIGroupResourceVerbs(
		t,
		filepath.Join(repoRoot(t), "packages", "identity", "install", "generated", "rbac", "role.yaml"),
		"identity.oracle.com",
		map[string][]string{
			"compartments":            {"create", "delete", "get", "list", "patch", "update", "watch"},
			"compartments/finalizers": {"update"},
			"compartments/status":     {"get", "patch", "update"},
		},
	)
}

func assertControllerHasEventRecorderRBAC(t *testing.T, controllerPath string) {
	t.Helper()

	assertFileContains(t, controllerPath, []string{
		`// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch`,
	})
	assertFileDoesNotContain(t, controllerPath, []string{
		`// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
	})
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

func assertAPIGroupResourceVerbs(t *testing.T, path string, apiGroup string, want map[string][]string) {
	t.Helper()

	role := loadClusterRole(t, path)
	got := collectAPIGroupResourceVerbs(role.Rules, apiGroup, want)

	for resource, wantVerbs := range want {
		gotVerbs := sortedKeys(got[resource])
		if !slices.Equal(gotVerbs, wantVerbs) {
			t.Fatalf("%s %s/%s verbs = %v, want %v", path, apiGroup, resource, gotVerbs, wantVerbs)
		}
	}
}

func collectAPIGroupResourceVerbs(rules []rbacv1.PolicyRule, apiGroup string, tracked map[string][]string) map[string]map[string]struct{} {
	got := make(map[string]map[string]struct{}, len(tracked))
	for _, rule := range rules {
		if !containsString(rule.APIGroups, apiGroup) {
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
