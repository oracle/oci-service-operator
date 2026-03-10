package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/oracle/oci-service-operator/pkg/validator"
	"github.com/oracle/oci-service-operator/pkg/validator/config"
)

func main() {
	opts := config.DefaultOptions()
	flag.StringVar(&opts.ProviderPath, "provider-path", opts.ProviderPath, "Path to the OSOK repository to analyze")
	flag.StringVar(&opts.AllowlistPath, "allowlist", opts.AllowlistPath, "Optional allowlist file")
	flag.StringVar(&opts.Format, "format", opts.Format, "Output format: table, json, or markdown")
	flag.StringVar(&opts.BaselinePath, "baseline", opts.BaselinePath, "Optional baseline report path to diff against")
	flag.StringVar(&opts.WriteBaseline, "write-baseline", opts.WriteBaseline, "Write the current report to the given path")
	flag.BoolVar(&opts.FailOnNew, "fail-on-new-actionable", opts.FailOnNew, "Exit with a non-zero status when new actionable gaps are found")
	flag.Parse()

	_, rendered, err := validator.Run(opts)
	if err != nil {
		if errors.Is(err, validator.ErrNewActionable) {
			fmt.Print(rendered)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "validator failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(rendered)
	os.Exit(0)
}
