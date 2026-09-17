/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/oracle/oci-service-operator/internal/integration/ocimockinventory"
)

func main() {
	root := flag.String("root", ".", "repository root")
	jsonOutput := flag.Bool("json", false, "emit the complete inventory as JSON")
	group := flag.String("group", "", "list resources in one group")
	checkImmediate := flag.Bool("check-immediate", false, "fail when an explicitly immediate resource lacks a typed mock scenario")
	checkLifecycle := flag.Bool("check-lifecycle", false, "fail when a lifecycle-polled resource lacks a typed mock scenario or formal row")
	flag.Parse()

	report, err := ocimockinventory.Audit(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OCI mock integration inventory failed: %v\n", err)
		os.Exit(1)
	}
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "encode OCI mock integration inventory: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Printf("Generated CRUD resources: %d\n", report.GeneratedCRUD)
	fmt.Printf("  synchronous CRUD: %d\n", report.SynchronousCRUD)
	fmt.Printf("    %s: %d\n", ocimockinventory.GroupImmediate, report.Groups[ocimockinventory.GroupImmediate])
	fmt.Printf("    %s: %d\n", ocimockinventory.GroupUnclassified, report.Groups[ocimockinventory.GroupUnclassified])
	fmt.Printf("    %s: %d\n", ocimockinventory.GroupLifecycle, report.Groups[ocimockinventory.GroupLifecycle])
	fmt.Printf("    %s: %d\n", ocimockinventory.GroupComposite, report.Groups[ocimockinventory.GroupComposite])
	fmt.Printf("  %s: %d\n", ocimockinventory.GroupWorkRequest, report.WorkRequestCRUD)
	fmt.Printf("Synchronous contracts: formal=%d runtime-overrides=%d typed-mock=%d\n", report.SynchronousFormalResources, report.SynchronousRuntimeOverrides, report.SynchronousMockIntegration)
	fmt.Printf("All generated CRUD contracts: formal=%d runtime-overrides=%d typed-mock=%d\n", report.FormalResources, report.RuntimeOverrides, report.MockIntegration)
	if *checkImmediate {
		missing := ocimockinventory.MissingMockIntegration(report, ocimockinventory.GroupImmediate)
		if len(missing) > 0 {
			for _, resource := range missing {
				fmt.Fprintf(os.Stderr, "missing immediate mock integration: %s/%s (%s)\n", resource.Service, resource.Kind, resource.PackagePath)
			}
			os.Exit(1)
		}
	}
	if *checkLifecycle {
		failed := false
		for _, resource := range ocimockinventory.MissingMockIntegration(report, ocimockinventory.GroupLifecycle) {
			fmt.Fprintf(os.Stderr, "missing lifecycle mock integration: %s/%s (%s)\n", resource.Service, resource.Kind, resource.PackagePath)
			failed = true
		}
		for _, resource := range ocimockinventory.MissingFormal(report, ocimockinventory.GroupLifecycle) {
			fmt.Fprintf(os.Stderr, "missing lifecycle formal row: %s/%s (%s)\n", resource.Service, resource.Kind, resource.PackagePath)
			failed = true
		}
		if failed {
			os.Exit(1)
		}
	}
	if *group == "" {
		return
	}
	for _, resource := range report.Resources {
		if string(resource.Group) == *group {
			fmt.Printf("%s/%s\tformal=%t runtime-override=%t mock-integration=%t\n", resource.Service, resource.Kind, resource.Formal, resource.RuntimeOverride, resource.MockIntegration)
		}
	}
}
