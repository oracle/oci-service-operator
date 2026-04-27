/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance

import generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"

// ExportSetClientForTest sets the OCI client on the service manager for unit testing.
func ExportSetClientForTest(m *ContainerInstanceServiceManager, c ContainerInstanceClientInterface) {
	m.ociClient = c
}

func ExportSetVnicClientForTest(m *ContainerInstanceServiceManager, c ContainerInstanceVnicClientInterface) {
	m.vnicClient = c
}

func ExportRuntimeSemanticsForTest() *generatedruntime.Semantics {
	return containerInstanceRuntimeSemantics()
}
