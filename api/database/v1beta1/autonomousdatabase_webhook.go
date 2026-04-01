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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var autonomousdatabaselog = logf.Log.WithName("autonomousdatabase-resource")

func (r *AutonomousDatabase) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-database-oracle-com-v1beta1-autonomousdatabase,mutating=true,failurePolicy=ignore,sideEffects=None,groups=database.oracle.com,resources=autonomousdatabases,verbs=create;update,versions=v1beta1,name=mautonomousdatabase.kb.io,admissionReviewVersions={v1}

var _ webhook.Defaulter = &AutonomousDatabase{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *AutonomousDatabase) Default() {
	autonomousdatabaselog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:path=/validate-database-oracle-com-v1beta1-autonomousdatabase,mutating=false,failurePolicy=fail,sideEffects=None,groups=database.oracle.com,resources=autonomousdatabases,verbs=create;update,versions=v1beta1,name=vautonomousdatabase.kb.io,admissionReviewVersions={v1}

var _ webhook.Validator = &AutonomousDatabase{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *AutonomousDatabase) ValidateCreate() (admission.Warnings, error) {
	autonomousdatabaselog.Info("validate create", "name", r.Name)

	autonomousdatabaselog.Info("We are validating the Create method for Autonomous Database")
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *AutonomousDatabase) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	autonomousdatabaselog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *AutonomousDatabase) ValidateDelete() (admission.Warnings, error) {
	autonomousdatabaselog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
