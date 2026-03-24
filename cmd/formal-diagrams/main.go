package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/oracle/oci-service-operator/internal/formal"
)

func main() {
	opts := formal.RenderOptions{}
	flag.StringVar(&opts.Root, "root", "formal", "Path to the repo-local formal scaffold")
	flag.StringVar(&opts.Service, "service", "", "Optional service filter")
	flag.Parse()

	report, err := formal.RenderDiagrams(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "formal-diagrams: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(report.String())
}
