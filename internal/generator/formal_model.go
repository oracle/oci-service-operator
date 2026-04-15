/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generator

import "github.com/oracle/oci-service-operator/internal/formal"

// FormalReferenceModel identifies the formal catalog entry backing a generated resource.
type FormalReferenceModel struct {
	Service string
	Slug    string
}

// FormalModel joins a generator model to one typed formal controller binding.
type FormalModel struct {
	Reference        FormalReferenceModel
	Binding          formal.ControllerBinding
	Diagrams         formal.DiagramFiles
	RuntimeLifecycle *formal.RuntimeLifecycleSpec
}
