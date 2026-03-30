/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatorcmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/oracle/oci-service-operator/internal/generator"
)

const DefaultConfigPath = "internal/generator/config/services.yaml"

type options struct {
	configPath string
	service    string
	all        bool
	outputRoot string
	overwrite  bool
	preserve   bool
}

// Main runs the user-facing generator CLI and returns a process exit code.
func Main(programName string, args []string, stdout io.Writer, stderr io.Writer) int {
	if err := run(context.Background(), programName, args, stdout, stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(stderr, "%s: %v\n", programName, err)
		return 1
	}

	return 0
}

func run(ctx context.Context, programName string, args []string, stdout io.Writer, stderr io.Writer) error {
	opts := options{}

	flagSet := flag.NewFlagSet(programName, flag.ContinueOnError)
	flagSet.SetOutput(stderr)
	flagSet.StringVar(&opts.configPath, "config", DefaultConfigPath, "Path to the OSOK generator config file.")
	flagSet.StringVar(&opts.service, "service", "", "Generate a single configured service.")
	flagSet.BoolVar(&opts.all, "all", false, "Generate all configured services.")
	flagSet.StringVar(&opts.outputRoot, "output-root", ".", "Root directory where generated outputs are written.")
	flagSet.BoolVar(&opts.overwrite, "overwrite", false, "Overwrite existing generated package files when the target directory already exists.")
	flagSet.BoolVar(&opts.preserve, "preserve-existing-spec-surface", false, "Preserve the current checked-in API/spec helper surface, including compatibility-locked checked-in artifacts, while regenerating other outputs.")
	if err := flagSet.Parse(args); err != nil {
		return err
	}

	return execute(ctx, opts, stdout)
}

func execute(ctx context.Context, opts options, stdout io.Writer) error {
	cfg, err := generator.LoadConfig(opts.configPath)
	if err != nil {
		return err
	}

	services, err := cfg.SelectServices(opts.service, opts.all)
	if err != nil {
		return err
	}
	if err := cfg.VerifyFormalInputs(); err != nil {
		return err
	}

	pipeline := generator.New()
	preserveRoot := ""
	if opts.preserve {
		preserveRoot = "."
	}
	result, err := pipeline.Generate(ctx, cfg, services, generator.Options{
		OutputRoot:                      opts.outputRoot,
		Overwrite:                       opts.overwrite,
		SkipExisting:                    opts.all && !opts.overwrite,
		PreserveExistingSpecSurfaceRoot: preserveRoot,
	})
	if err != nil {
		return err
	}

	for _, generated := range result.Generated {
		fmt.Fprintf(stdout, "generated service=%s group=%s resources=%d output=%s\n", generated.Service, generated.Group, generated.ResourceCount, generated.OutputDir)
	}
	for _, skipped := range result.Skipped {
		fmt.Fprintf(stdout, "skipped service=%s group=%s reason=%s\n", skipped.Service, skipped.Group, skipped.Reason)
	}

	return nil
}
