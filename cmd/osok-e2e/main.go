/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/oracle/oci-service-operator/internal/e2e/lifecycle"
)

type variablesFlag map[string]string

func (v variablesFlag) String() string {
	return "NAME=value"
}

func (v variablesFlag) Set(value string) error {
	name, resolved, found := strings.Cut(value, "=")
	if !found || strings.TrimSpace(name) == "" {
		return fmt.Errorf("expected NAME=value")
	}
	v[name] = resolved
	return nil
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "lifecycle" {
		fmt.Fprintln(os.Stderr, "usage: osok-e2e lifecycle --scenario PATH [options]")
		os.Exit(2)
	}
	flags := flag.NewFlagSet("lifecycle", flag.ExitOnError)
	scenario := flags.String("scenario", "", "path to scenario.yaml")
	artifacts := flags.String("artifacts-dir", "", "directory for rendered manifests and result.json")
	kubeconfig := flags.String("kubeconfig", "", "kubeconfig path; defaults to kubectl configuration")
	kubectl := flags.String("kubectl", "kubectl", "kubectl binary")
	expectedService := flags.String("expected-service", "", "require the scenario to target this installed service")
	variables := variablesFlag{}
	flags.Var(variables, "var", "manifest variable as NAME=value; may be repeated")
	if err := flags.Parse(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *scenario == "" {
		fmt.Fprintln(os.Stderr, "--scenario is required")
		os.Exit(2)
	}

	result, err := lifecycle.Run(context.Background(), lifecycle.RunOptions{
		ScenarioPath:    *scenario,
		ArtifactsDir:    *artifacts,
		Kubeconfig:      *kubeconfig,
		KubectlBinary:   *kubectl,
		ExpectedService: *expectedService,
		Variables:       variables,
	})
	encoded, encodeErr := json.MarshalIndent(result, "", "  ")
	if encodeErr == nil {
		fmt.Println(string(encoded))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
