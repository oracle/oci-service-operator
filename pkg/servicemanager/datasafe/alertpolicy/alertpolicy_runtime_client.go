/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package alertpolicy

import (
	"fmt"

	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/datasafe/runtimecommon"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerAlertPolicyRuntimeHooksMutator(func(manager *AlertPolicyServiceManager, hooks *AlertPolicyRuntimeHooks) {
		client, initErr := newAlertPolicyWorkRequestClient(manager)
		applyAlertPolicyRuntimeHooks(hooks, client, initErr)
	})
}

func newAlertPolicyWorkRequestClient(manager *AlertPolicyServiceManager) (runtimecommon.WorkRequestClient, error) {
	if manager == nil {
		return nil, fmt.Errorf("AlertPolicy service manager is nil")
	}
	client, err := datasafesdk.NewDataSafeClientWithConfigurationProvider(manager.Provider)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func applyAlertPolicyRuntimeHooks(hooks *AlertPolicyRuntimeHooks, client runtimecommon.WorkRequestClient, initErr error) {
	if hooks == nil {
		return
	}
	hooks.Semantics = reviewedAlertPolicyRuntimeSemantics()
	runtimecommon.ConfigureWorkRequest(&hooks.Async, client, initErr, "AlertPolicy")
}

func reviewedAlertPolicyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "datasafe",
		FormalSlug:    "alertpolicy",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "workrequest",
			Runtime:              "generatedruntime",
			FormalClassification: "workrequest",
			WorkRequest: &generatedruntime.WorkRequestSemantics{
				Source: "service-sdk",
				Phases: []string{"create", "update", "delete"},
			},
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(datasafesdk.AlertPolicyLifecycleStateCreating)},
			UpdatingStates:     []string{string(datasafesdk.AlertPolicyLifecycleStateUpdating)},
			ActiveStates:       []string{string(datasafesdk.AlertPolicyLifecycleStateActive)},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(datasafesdk.AlertPolicyLifecycleStateDeleting)},
			TerminalStates: []string{string(datasafesdk.AlertPolicyLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"displayName", "description", "severity", "freeformTags", "definedTags"},
			ForceNew:      []string{"alertPolicyType", "compartmentId", "alertPolicyRuleDetails"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetAlertPolicy"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> GetAlertPolicy"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "GetWorkRequest -> confirm-delete"},
	}
}
