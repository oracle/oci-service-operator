/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package budget

import (
	"context"
	"strings"

	budgetv1beta1 "github.com/oracle/oci-service-operator/api/budget/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerBudgetRuntimeHooksMutator(func(_ *BudgetServiceManager, hooks *BudgetRuntimeHooks) {
		applyBudgetRuntimeHooks(hooks)
	})
}

func applyBudgetRuntimeHooks(hooks *BudgetRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Identity.GuardExistingBeforeCreate = guardBudgetExistingBeforeCreate
}

func guardBudgetExistingBeforeCreate(
	_ context.Context,
	resource *budgetv1beta1.Budget,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}
