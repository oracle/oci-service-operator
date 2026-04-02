package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/oracle/oci-service-operator/internal/validator"
	"github.com/oracle/oci-service-operator/internal/validator/config"
)

func main() {
	opts := config.DefaultOptions()
	flag.StringVar(&opts.ProviderPath, "provider-path", opts.ProviderPath, "Path to the OSOK repository to analyze")
	flag.StringVar(&opts.AllowlistPath, "allowlist", opts.AllowlistPath, "Optional allowlist file")
	flag.StringVar(&opts.Service, "service", opts.Service, "Optional service filter (for example: core, identity, psql)")
	flag.StringVar(&opts.Format, "format", opts.Format, "Output format: table, json, or markdown")
	flag.StringVar(&opts.BaselinePath, "baseline", opts.BaselinePath, "Optional baseline report path to diff against")
	flag.StringVar(&opts.WriteBaseline, "write-baseline", opts.WriteBaseline, "Write the current report to the given path")
	flag.BoolVar(&opts.FailOnNew, "fail-on-new-actionable", opts.FailOnNew, "Exit with a non-zero status when new actionable gaps are found")
	flag.StringVar(&opts.UpgradeFrom, "upgrade-from", opts.UpgradeFrom, "Generate an SDK diff starting from this version")
	flag.StringVar(&opts.UpgradeTo, "upgrade-to", opts.UpgradeTo, "Generate an SDK diff ending at this version")
	flag.Parse()

	if opts.WantsUpgrade() {
		if strings.TrimSpace(opts.UpgradeFrom) == "" || strings.TrimSpace(opts.UpgradeTo) == "" {
			fmt.Fprintln(os.Stderr, "--upgrade-from and --upgrade-to must both be provided when running in upgrade mode")
			os.Exit(2)
		}
		_, rendered, err := validator.RunUpgrade(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "upgrade analysis failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(rendered)
		os.Exit(0)
	}

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
