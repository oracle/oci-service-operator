package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/oracle/oci-service-operator/internal/formal"
)

func main() {
	root := flag.String("root", "formal", "Path to the repo-local formal scaffold")
	flag.Parse()

	report, err := formal.Verify(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "formal-verify: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(report.String())
}
