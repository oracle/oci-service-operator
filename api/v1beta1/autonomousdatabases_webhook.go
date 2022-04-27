/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var autonomousdatabaseslog = logf.Log.WithName("autonomousdatabases-resource")

func (r *AutonomousDatabases) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-oci-oci-v1beta1-autonomousdatabases,mutating=true,failurePolicy=ignore,sideEffects=None,groups=oci.oracle.com,resources=autonomousdatabases,verbs=create;update,versions=v1,name=mautonomousdatabases.kb.io,admissionReviewVersions={v1}

var _ webhook.Defaulter = &AutonomousDatabases{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AutonomousDatabases) Default() {
	autonomousdatabaseslog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-oci-oci-v1beta1-autonomousdatabases,mutating=false,failurePolicy=fail,sideEffects=None,groups=oci.oracle.com,resources=autonomousdatabases,verbs=create;update,versions=v1,name=vautonomousdatabases.kb.io,admissionReviewVersions={v1}

var _ webhook.Validator = &AutonomousDatabases{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AutonomousDatabases) ValidateCreate() error {
	autonomousdatabaseslog.Info("validate create", "name", r.Name)

	autonomousdatabaseslog.Info("We are validating the Create method for Autonomous Databases")
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AutonomousDatabases) ValidateUpdate(old runtime.Object) error {
	autonomousdatabaseslog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AutonomousDatabases) ValidateDelete() error {
	autonomousdatabaseslog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
