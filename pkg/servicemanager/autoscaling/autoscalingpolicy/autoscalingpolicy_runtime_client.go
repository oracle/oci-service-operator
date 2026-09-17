/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package autoscalingpolicy

import (
	"context"

	autoscalingv1beta1 "github.com/oracle/oci-service-operator/api/autoscaling/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	registerAutoScalingPolicyRuntimeHooksMutator(func(_ *AutoScalingPolicyServiceManager, hooks *AutoScalingPolicyRuntimeHooks) {
		hooks.Semantics = autoScalingPolicyRuntimeSemantics()
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapAutoScalingPolicyStateFreeClient)
	})
}

func autoScalingPolicyRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "autoscaling", FormalSlug: "autoscalingpolicy",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Delete: generatedruntime.DeleteSemantics{Policy: "required", TerminalStates: []string{"DELETED"}},
		List:   &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"displayName", "policyType"}},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{"capacity", "displayName", "isEnabled", "executionSchedule", "resourceAction", "rules"},
			ForceNew: []string{"autoScalingConfigurationId", "policyType"},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

type autoScalingPolicyStateFreeClient struct {
	delegate AutoScalingPolicyServiceClient
}

func wrapAutoScalingPolicyStateFreeClient(delegate AutoScalingPolicyServiceClient) AutoScalingPolicyServiceClient {
	return autoScalingPolicyStateFreeClient{delegate: delegate}
}

func (c autoScalingPolicyStateFreeClient) CreateOrUpdate(ctx context.Context, resource *autoscalingv1beta1.AutoScalingPolicy, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err == nil && response.IsSuccessful && response.ShouldRequeue && resource != nil && resource.Status.OsokStatus.Ocid != "" {
		now := metav1.Now()
		resource.Status.OsokStatus.Message = "OCI autoscaling policy is active"
		resource.Status.OsokStatus.Reason = string(shared.Active)
		resource.Status.OsokStatus.UpdatedAt = &now
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, corev1.ConditionTrue, "", resource.Status.OsokStatus.Message, loggerutil.OSOKLogger{})
		response.ShouldRequeue = false
		response.RequeueDuration = 0
	}
	return response, err
}

func (c autoScalingPolicyStateFreeClient) Delete(ctx context.Context, resource *autoscalingv1beta1.AutoScalingPolicy) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}
