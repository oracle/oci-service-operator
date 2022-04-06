/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package authhelper

import (
	"github.com/oracle/oci-go-sdk/v65/common"

	"github.com/oracle/oci-service-operator/pkg/config"
)

type AuthProvider interface {
	GetAuthProvider(cfg config.OsokConfig) (common.ConfigurationProvider, error)
}
