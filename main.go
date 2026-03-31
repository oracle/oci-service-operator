/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package main

import (
	"fmt"
	"os"

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
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/oracle/oci-service-operator/pkg/authhelper"
	"github.com/oracle/oci-service-operator/pkg/config"
	"github.com/oracle/oci-service-operator/pkg/credhelper/kubesecret"
	"github.com/oracle/oci-service-operator/pkg/metrics"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"github.com/oracle/oci-service-operator/pkg/util"
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
	if err := run(); err != nil {
		setupLog.ErrorLog(err, "manager exited")
		os.Exit(1)
	}
}

func run() error {
	go_ensurefips.Compliant()
	common.EnableInstanceMetadataServiceLookup()

	startup := parseStartupFlags()

	options, err := resolveManagerOptions(startup)
	if err != nil {
		return err
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		return fmt.Errorf("unable to start manager: %w", err)
	}

	if startup.initOSOKResources {
		util.InitOSOK(mgr.GetConfig(), loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("initOSOK")})
	}

	runtimeDeps, err := buildRuntimeDeps(mgr)
	if err != nil {
		return err
	}

	if err := setupRuntimeRegistrations(mgr, runtimeDeps); err != nil {
		return err
	}

	// +kubebuilder:scaffold:builder

	if err := addManagerChecks(mgr); err != nil {
		return err
	}

	setupLog.InfoLog("starting manager")
	return mgr.Start(ctrl.SetupSignalHandler())
}

func buildRuntimeDeps(mgr ctrl.Manager) (servicemanager.RuntimeDeps, error) {
	setupLog.InfoLog("Getting the config details")
	configLogger := loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("setup").WithName("config")}
	osokCfg := config.GetConfigDetails(configLogger)

	authConfigProvider := &authhelper.AuthConfigProvider{Log: configLogger}
	provider, err := authConfigProvider.GetAuthProvider(osokCfg)
	if err != nil {
		return servicemanager.RuntimeDeps{}, fmt.Errorf("unable to get the oci configuration provider: %w", err)
	}

	metricsClient := metrics.Init("osok", loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("metrics")})
	credClient := &kubesecret.KubeSecretClient{
		Client:  mgr.GetClient(),
		Log:     loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("credential-helper").WithName("KubeSecretClient")},
		Metrics: metricsClient,
	}

	return servicemanager.RuntimeDeps{
		Provider:         provider,
		CredentialClient: credClient,
		Scheme:           mgr.GetScheme(),
		Metrics:          metricsClient,
	}, nil
}

func setupRuntimeRegistrations(mgr ctrl.Manager, runtimeDeps servicemanager.RuntimeDeps) error {
	registrationContext := registrations.NewContext(mgr, runtimeDeps)
	if err := setupControllerRegistrations(registrationContext); err != nil {
		return err
	}
	return setupExplicitWebhooks(mgr)
}

func setupControllerRegistrations(registrationContext registrations.Context) error {
	for _, registration := range registrations.All() {
		if err := registration.SetupWithManager(registrationContext); err != nil {
			return fmt.Errorf("unable to create controller registration for group %q: %w", registration.Group, err)
		}
	}
	return nil
}

func setupExplicitWebhooks(mgr ctrl.Manager) error {
	for _, webhook := range registrations.ExplicitWebhooks() {
		if err := webhook.SetupWithManager(mgr); err != nil {
			return fmt.Errorf("unable to create webhook %q: %w", webhook.Name, err)
		}
	}
	return nil
}

func addManagerChecks(mgr ctrl.Manager) error {
	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}
	return nil
}
