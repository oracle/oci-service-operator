/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package model

import (
	"context"
	"strings"

	generativeaiv1beta1 "github.com/oracle/oci-service-operator/api/generativeai/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerModelRuntimeHooksMutator(func(_ *ModelServiceManager, hooks *ModelRuntimeHooks) {
		applyModelRuntimeHooks(hooks)
	})
}

func applyModelRuntimeHooks(hooks *ModelRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Identity.GuardExistingBeforeCreate = guardModelExistingBeforeCreate
	hooks.TrackedRecreate.ClearTrackedIdentity = clearModelIdentity
}

func guardModelExistingBeforeCreate(
	_ context.Context,
	resource *generativeaiv1beta1.Model,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func clearModelIdentity(resource *generativeaiv1beta1.Model) {
	if resource == nil {
		return
	}
	resource.Status.Id = ""
	resource.Status.OsokStatus.Ocid = shared.OCID("")
}
