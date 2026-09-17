/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package targetresponderrecipe

import (
	"context"

	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	registerTargetResponderRecipeRuntimeHooksMutator(func(_ *TargetResponderRecipeServiceManager, hooks *TargetResponderRecipeRuntimeHooks) {
		hooks.Semantics = newTargetResponderRecipeRuntimeSemantics()
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapTargetResponderRecipeStateFreeClient)
	})
}

func newTargetResponderRecipeRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "cloudguard", FormalSlug: "targetresponderrecipe",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Delete:         generatedruntime.DeleteSemantics{Policy: "required", TerminalStates: []string{"DELETED"}},
		List:           &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"targetId", "compartmentId", "responderRecipeId"}},
		Mutation:       generatedruntime.MutationSemantics{Mutable: []string{"responderRules"}, ForceNew: []string{"targetId", "compartmentId", "responderRecipeId"}, ConflictsWith: map[string][]string{}},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

type targetResponderRecipeStateFreeClient struct {
	delegate TargetResponderRecipeServiceClient
}

func wrapTargetResponderRecipeStateFreeClient(delegate TargetResponderRecipeServiceClient) TargetResponderRecipeServiceClient {
	return targetResponderRecipeStateFreeClient{delegate: delegate}
}

func (c targetResponderRecipeStateFreeClient) CreateOrUpdate(ctx context.Context, resource *cloudguardv1beta1.TargetResponderRecipe, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err == nil && response.IsSuccessful && response.ShouldRequeue && resource != nil && resource.Status.OsokStatus.Ocid != "" {
		now := metav1.Now()
		resource.Status.OsokStatus.Message = "OCI target responder recipe is active"
		resource.Status.OsokStatus.Reason = string(shared.Active)
		resource.Status.OsokStatus.UpdatedAt = &now
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", resource.Status.OsokStatus.Message, loggerutil.OSOKLogger{})
		response.ShouldRequeue = false
		response.RequeueDuration = 0
	}
	return response, err
}

func (c targetResponderRecipeStateFreeClient) Delete(ctx context.Context, resource *cloudguardv1beta1.TargetResponderRecipe) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}
