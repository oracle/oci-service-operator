/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"flag"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-service-operator/go_ensurefips"
	"github.com/oracle/oci-service-operator/internal/registrations"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup")}
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	for _, registration := range registrations.All() {
		utilruntime.Must(registration.AddToScheme(scheme))
	}
	// +kubebuilder:scaffold:scheme
}

func main() {
	// Check for fips compliance
	go_ensurefips.Compliant()

	// Allow OCI go sdk to use instance metadata service for region lookup
	common.EnableInstanceMetadataServiceLookup()

	var configFile string
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var initOSOKResources bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&initOSOKResources, "init-osok-resources", false,
		"Install OSOK prerequisites like CRDs and Webhooks at manager bootup")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	options := mustManagerOptions(configFile, metricsAddr, probeAddr, enableLeaderElection)
	mgr := mustNewManager(options)

	maybeInitOSOKResources(mgr, initOSOKResources)
	mustSetupRegistrations(mgr)
	mustSetupManualWebhooks(mgr)

	// +kubebuilder:scaffold:builder

	mustAddHealthChecks(mgr)
	mustStartManager(mgr)
}
