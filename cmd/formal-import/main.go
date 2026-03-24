package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/oracle/oci-service-operator/internal/formal"
)

func main() {
	opts := formal.ImportOptions{}
	flag.StringVar(&opts.Root, "root", "formal", "Path to the repo-local formal scaffold")
	flag.StringVar(&opts.ProviderPath, "provider-path", "", "Path to the pinned terraform-provider-oci checkout")
	flag.StringVar(&opts.ProviderRevision, "provider-revision", "", "Optional provider revision override when the source tree is not a git checkout")
	flag.StringVar(&opts.Service, "service", "", "Optional service filter for import refresh")
	flag.StringVar(&opts.SourceName, "source-name", "terraform-provider-oci", "Source name recorded in formal/sources.lock")
	flag.Parse()

	report, err := formal.Import(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "formal-import: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(report.String())
}
