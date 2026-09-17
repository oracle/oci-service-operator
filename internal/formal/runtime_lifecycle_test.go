/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package formal

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestCheckedInRuntimeLifecycleRecordsPackageLocalUpdateHooks(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		service string
		slug    string
		primary string
		handled string
	}{
		{service: "datasafe", slug: "attributeset", primary: "UpdateAttributeSet", handled: "ChangeAttributeSetCompartment"},
		{service: "datasafe", slug: "securitypolicyconfig", primary: "UpdateSecurityPolicyConfig", handled: "ChangeSecurityPolicyConfigCompartment"},
		{service: "networkloadbalancer", slug: "networkloadbalancer", primary: "UpdateNetworkLoadBalancer", handled: "UpdateNetworkSecurityGroups"},
		{service: "recovery", slug: "protecteddatabase", primary: "UpdateProtectedDatabase", handled: "ChangeProtectedDatabaseCompartment"},
		{service: "recovery", slug: "protecteddatabase", primary: "UpdateProtectedDatabase", handled: "ChangeProtectedDatabaseSubscription"},
		{service: "recovery", slug: "recoveryservicesubnet", primary: "UpdateRecoveryServiceSubnet", handled: "ChangeRecoveryServiceSubnetCompartment"},
		{service: "waas", slug: "waaspolicy", primary: "UpdateWaasPolicy", handled: "ChangeWaasPolicyCompartment"},
		{service: "waf", slug: "webappfirewallpolicy", primary: "UpdateWebAppFirewallPolicy", handled: "ChangeWebAppFirewallPolicyCompartment"},
	} {
		t.Run(tc.service+"/"+tc.slug+"/"+tc.handled, func(t *testing.T) {
			t.Parallel()

			spec, imported := loadCheckedInRuntimeLifecycle(t, tc.service, tc.slug)
			got := operationNames(EffectiveRuntimeLifecycleUpdateOperations(spec, imported.Operations.Update))
			if !reflect.DeepEqual(got, []string{tc.primary}) {
				t.Fatalf("effective generated-runtime update operations = %v, want only primary %q", got, tc.primary)
			}
			if spec.RepoAuthored == nil || spec.RepoAuthored.Hooks == nil || !hookHelpersContain(spec.RepoAuthored.Hooks.Update, tc.handled) {
				t.Fatalf("repo-authored update hooks = %#v, want package-local helper %q", spec.RepoAuthored, tc.handled)
			}
		})
	}
}

func TestCheckedInRuntimeLifecycleFiltersProviderOnlyCreateDeleteOperations(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		service    string
		slug       string
		wantCreate []string
		wantDelete []string
	}{
		{service: "opsi", slug: "exadatainsight", wantCreate: []string{"CreateExadataInsight"}, wantDelete: []string{"DeleteExadataInsight"}},
		{service: "opsi", slug: "hostinsight", wantCreate: []string{"CreateHostInsight"}, wantDelete: []string{"DeleteHostInsight"}},
		{service: "recovery", slug: "protecteddatabase", wantCreate: []string{"CreateProtectedDatabase"}, wantDelete: []string{}},
	} {
		t.Run(tc.service+"/"+tc.slug, func(t *testing.T) {
			t.Parallel()

			spec, imported := loadCheckedInRuntimeLifecycle(t, tc.service, tc.slug)
			if got := operationNames(EffectiveRuntimeLifecycleCreateOperations(spec, imported.Operations.Create)); !reflect.DeepEqual(got, tc.wantCreate) {
				t.Fatalf("effective create operations = %v, want %v", got, tc.wantCreate)
			}
			if got := operationNames(EffectiveRuntimeLifecycleDeleteOperations(spec, imported.Operations.Delete)); !reflect.DeepEqual(got, tc.wantDelete) {
				t.Fatalf("effective delete operations = %v, want %v", got, tc.wantDelete)
			}
		})
	}
}

func loadCheckedInRuntimeLifecycle(t *testing.T, service, slug string) (*RuntimeLifecycleSpec, importFile) {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "formal"))
	spec, err := LoadRuntimeLifecycle(filepath.Join(root, "controllers", service, slug, "diagrams", "runtime-lifecycle.yaml"))
	if err != nil {
		t.Fatalf("LoadRuntimeLifecycle(%s/%s) error = %v", service, slug, err)
	}
	imported, err := loadImport(filepath.Join(root, "imports", service, slug+".json"))
	if err != nil {
		t.Fatalf("loadImport(%s/%s) error = %v", service, slug, err)
	}
	return &spec, imported
}

func operationNames(operations []OperationBinding) []string {
	names := make([]string, 0, len(operations))
	for _, operation := range operations {
		names = append(names, operation.Operation)
	}
	return names
}

func hookHelpersContain(hooks []hook, want string) bool {
	for _, hook := range hooks {
		if hook.Helper == want {
			return true
		}
	}
	return false
}
