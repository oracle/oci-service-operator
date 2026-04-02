package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/oracle/oci-service-operator/internal/formalscaffold"
)

func main() {
	opts := formalscaffold.Options{}
	flag.StringVar(&opts.Root, "root", "formal", "Path to the repo-local formal scaffold")
	flag.StringVar(&opts.ConfigPath, "config", "internal/generator/config/services.yaml", "Path to the generator config that defines the published default-active API surface")
	flag.StringVar(&opts.ProviderPath, "provider-path", "", "Path to the pinned terraform-provider-oci checkout")
	flag.Parse()

	report, err := formalscaffold.Generate(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "formal-scaffold: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(report.String())
}
