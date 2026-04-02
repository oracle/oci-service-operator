package formalscaffold

import (
	"os/exec"
	"testing"
)

func requirePlantUML(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("plantuml"); err != nil {
		t.Skip("plantuml binary not found in PATH")
	}
}
