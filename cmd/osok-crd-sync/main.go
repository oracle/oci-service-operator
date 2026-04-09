package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/oracle/oci-service-operator/internal/crdsync"
)

func main() {
	var kustomizationPath string
	var basesDir string

	flag.StringVar(&kustomizationPath, "kustomization", "config/crd/kustomization.yaml", "Path to the shared CRD kustomization file.")
	flag.StringVar(&basesDir, "bases-dir", "config/crd/bases", "Path to the directory containing generated CRD base manifests.")
	flag.Parse()

	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "osok-crd-sync: unexpected positional arguments")
		os.Exit(2)
	}

	if err := crdsync.SyncFile(kustomizationPath, basesDir); err != nil {
		fmt.Fprintf(os.Stderr, "osok-crd-sync: %v\n", err)
		os.Exit(1)
	}
}
