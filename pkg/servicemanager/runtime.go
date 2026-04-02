/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package servicemanager

import (
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-service-operator/pkg/credhelper"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"k8s.io/apimachinery/pkg/runtime"
)

// RuntimeDeps is the shared constructor contract for service-manager runtime inputs.
// Optional extras such as Metrics may be nil when a service manager does not need them.
type RuntimeDeps struct {
	Provider         common.ConfigurationProvider
	CredentialClient credhelper.CredentialClient
	Scheme           *runtime.Scheme
	Log              loggerutil.OSOKLogger
	Metrics          *metrics.Metrics
}

// Factory constructs an OSOK service manager from the shared runtime dependency contract.
type Factory func(RuntimeDeps) OSOKServiceManager

// WithLog returns a copy of the runtime deps with a resource-specific logger applied.
func (d RuntimeDeps) WithLog(log loggerutil.OSOKLogger) RuntimeDeps {
	d.Log = log
	return d
}

// WithScheme returns a copy of the runtime deps with the manager scheme applied.
func (d RuntimeDeps) WithScheme(scheme *runtime.Scheme) RuntimeDeps {
	d.Scheme = scheme
	return d
}
