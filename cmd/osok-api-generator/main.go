/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/oracle/oci-service-operator/internal/generator"
)

func main() {
	var (
		configPath string
		service    string
		all        bool
		outputRoot string
		overwrite  bool
		preserve   bool
	)

	flag.StringVar(&configPath, "config", "internal/generator/config/services.yaml", "Path to the OSOK API generator config file.")
	flag.StringVar(&service, "service", "", "Generate a single configured service.")
	flag.BoolVar(&all, "all", false, "Generate all configured services.")
	flag.StringVar(&outputRoot, "output-root", ".", "Root directory where generated API packages are written.")
	flag.BoolVar(&overwrite, "overwrite", false, "Overwrite existing generated package files when the target directory already exists.")
	flag.BoolVar(&preserve, "preserve-existing-spec-surface", false, "Preserve the current checked-in spec/helper surface for existing API packages while regenerating status/read-model outputs.")
	flag.Parse()

	if err := run(context.Background(), configPath, service, all, outputRoot, overwrite, preserve); err != nil {
		fmt.Fprintf(os.Stderr, "osok-api-generator: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configPath string, service string, all bool, outputRoot string, overwrite bool, preserve bool) error {
	cfg, err := generator.LoadConfig(configPath)
	if err != nil {
		return err
	}

	services, err := cfg.SelectServices(service, all)
	if err != nil {
		return err
	}

	pipeline := generator.New()
	preserveRoot := ""
	if preserve {
		preserveRoot = "."
	}
	result, err := pipeline.Generate(ctx, cfg, services, generator.Options{
		OutputRoot:                      outputRoot,
		Overwrite:                       overwrite,
		SkipExisting:                    all && !overwrite,
		PreserveExistingSpecSurfaceRoot: preserveRoot,
	})
	if err != nil {
		return err
	}

	for _, generated := range result.Generated {
		fmt.Printf("generated service=%s group=%s resources=%d output=%s\n", generated.Service, generated.Group, generated.ResourceCount, generated.OutputDir)
	}
	for _, skipped := range result.Skipped {
		fmt.Printf("skipped service=%s group=%s reason=%s\n", skipped.Service, skipped.Group, skipped.Reason)
	}

	return nil
}
