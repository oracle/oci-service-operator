/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import (
	"path/filepath"
	"testing"
)

func TestCheckedInDatabaseAutonomousDatabaseRBACMatchesReadOnlySecretSemantics(t *testing.T) {
	controllerPath := filepath.Join(repoRoot(t), "controllers", "database", "autonomousdatabase_controller.go")
	assertFileContains(t, controllerPath, []string{
		`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch`,
		`// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch`,
	})
	assertFileDoesNotContain(t, controllerPath, []string{
		`// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete`,
		`// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete`,
	})
}
