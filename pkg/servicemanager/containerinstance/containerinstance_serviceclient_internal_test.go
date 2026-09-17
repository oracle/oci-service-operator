/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package containerinstance

import (
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	containerinstancessdk "github.com/oracle/oci-go-sdk/v65/containerinstances"
	containerinstancesv1beta1 "github.com/oracle/oci-service-operator/api/containerinstances/v1beta1"
)

func TestValidateContainerMatchesDesiredIgnoresProviderDefaultSecurityContext(t *testing.T) {
	t.Parallel()

	desired := containerinstancesv1beta1.ContainerInstanceContainer{
		ImageUrl:    "docker.io/library/busybox:1.36.1",
		DisplayName: "worker",
		Command:     []string{"/bin/sh", "-c"},
		Arguments:   []string{"sleep 3600"},
	}
	observed := containerinstancessdk.Container{
		ImageUrl:    common.String(desired.ImageUrl),
		DisplayName: common.String(desired.DisplayName),
		Command:     desired.Command,
		Arguments:   desired.Arguments,
		SecurityContext: containerinstancessdk.LinuxSecurityContext{
			IsNonRootUserCheckEnabled: common.Bool(false),
			IsRootFileSystemReadonly:  common.Bool(false),
		},
	}

	if err := validateContainerMatchesDesired(0, desired, observed); err != nil {
		t.Fatalf("validateContainerMatchesDesired() error = %v", err)
	}
}
