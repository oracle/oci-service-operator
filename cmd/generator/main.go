/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"os"

	"github.com/oracle/oci-service-operator/internal/generatorcmd"
)

func main() {
	os.Exit(generatorcmd.Main("generator", os.Args[1:], os.Stdout, os.Stderr))
}
