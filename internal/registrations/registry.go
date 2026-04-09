/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package registrations

import (
	"sort"

	"github.com/oracle/oci-service-operator/pkg/core"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GroupRegistration standardizes scheme/controller registration for one API group.
type GroupRegistration struct {
	Group            string
	AddToScheme      func(*runtime.Scheme) error
	SetupWithManager func(Context) error
}

// WebhookRegistration keeps webhook setup explicit and separate from group runtime registration.
type WebhookRegistration struct {
	Group            string
	Name             string
	SetupWithManager func(ctrl.Manager) error
}

// Context carries the manager and shared runtime deps needed to wire a group.
type Context struct {
	Manager            ctrl.Manager
	Client             client.Client
	Scheme             *runtime.Scheme
	EventRecorderFor   func(string) record.EventRecorder
	ServiceManagerDeps servicemanager.RuntimeDeps
}

var generatedGroupRegistrations []GroupRegistration

// NewContext captures the shared manager-backed dependencies for registration setup.
func NewContext(mgr ctrl.Manager, deps servicemanager.RuntimeDeps) Context {
	scheme := mgr.GetScheme()
	return Context{
		Manager:            mgr,
		Client:             mgr.GetClient(),
		Scheme:             scheme,
		EventRecorderFor:   mgr.GetEventRecorderFor,
		ServiceManagerDeps: deps.WithScheme(scheme),
	}
}

// NewBaseReconciler applies the shared runtime deps contract to one controller resource.
func NewBaseReconciler(ctx Context, component string, factory servicemanager.Factory) *core.BaseReconciler {
	serviceManagerDeps := ctx.ServiceManagerDeps.WithScheme(ctx.Scheme).WithLog(
		loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName(component)},
	)

	var recorder record.EventRecorder
	if ctx.EventRecorderFor != nil {
		recorder = ctx.EventRecorderFor(component)
	}

	return &core.BaseReconciler{
		Client:             ctx.Client,
		OSOKServiceManager: factory(serviceManagerDeps),
		Finalizer:          core.NewBaseFinalizer(ctx.Client, ctrl.Log),
		Log:                loggerutil.OSOKLogger{Logger: ctrl.Log.WithName("controllers").WithName(component)},
		Metrics:            serviceManagerDeps.Metrics,
		Recorder:           recorder,
		Scheme:             ctx.Scheme,
	}
}

func registerGeneratedGroup(registration GroupRegistration) {
	generatedGroupRegistrations = append(generatedGroupRegistrations, registration)
}

// All returns the currently wired group registrations in startup order.
func All() []GroupRegistration {
	registrations := append([]GroupRegistration(nil), manualGroupRegistrations...)
	if len(generatedGroupRegistrations) == 0 {
		return registrations
	}

	generated := append([]GroupRegistration(nil), generatedGroupRegistrations...)
	sort.Slice(generated, func(i, j int) bool {
		return generated[i].Group < generated[j].Group
	})
	return appendUniqueGroupRegistrations(registrations, generated...)
}

// ManualWebhooks returns explicit webhook hooks that stay outside generated runtime registration.
func ManualWebhooks() []WebhookRegistration {
	return append([]WebhookRegistration(nil), manualWebhookRegistrations...)
}

// ByGroup returns the registration for one group when it exists.
func ByGroup(group string) (GroupRegistration, bool) {
	for _, registration := range All() {
		if registration.Group == group {
			return registration, true
		}
	}
	return GroupRegistration{}, false
}

// ManualWebhooksByGroup returns explicit webhook hooks for one API group.
func ManualWebhooksByGroup(group string) []WebhookRegistration {
	if group == "" {
		return nil
	}

	webhooks := make([]WebhookRegistration, 0, len(manualWebhookRegistrations))
	for _, webhook := range manualWebhookRegistrations {
		if webhook.Group != group {
			continue
		}
		webhooks = append(webhooks, webhook)
	}
	return webhooks
}

func appendUniqueGroupRegistrations(existing []GroupRegistration, extras ...GroupRegistration) []GroupRegistration {
	seen := make(map[string]struct{}, len(existing)+len(extras))
	for _, registration := range existing {
		seen[registration.Group] = struct{}{}
	}
	for _, registration := range extras {
		if _, ok := seen[registration.Group]; ok {
			continue
		}
		seen[registration.Group] = struct{}{}
		existing = append(existing, registration)
	}
	return existing
}
