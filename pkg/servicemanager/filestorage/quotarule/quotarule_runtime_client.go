/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package quotarule

import (
	"context"

	filestoragev1beta1 "github.com/oracle/oci-service-operator/api/filestorage/v1beta1"
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
	registerQuotaRuleRuntimeHooksMutator(func(_ *QuotaRuleServiceManager, hooks *QuotaRuleRuntimeHooks) {
		hooks.Semantics = quotaRuleRuntimeSemantics()
		hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapQuotaRuleStateFreeClient)
	})
}

func quotaRuleRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "filestorage", FormalSlug: "quotarule",
		Async:            &generatedruntime.AsyncSemantics{Strategy: "none", Runtime: "generatedruntime", FormalClassification: "none"},
		StatusProjection: "required", SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Delete: generatedruntime.DeleteSemantics{Policy: "required", TerminalStates: []string{"DELETED"}},
		List:   &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"fileSystemId", "principalType", "principalId", "isHardQuota"}},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{"displayName", "quotaLimitInGigabytes"},
			ForceNew: []string{"fileSystemId", "principalType", "principalId", "isHardQuota"},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

type quotaRuleStateFreeClient struct{ delegate QuotaRuleServiceClient }

func wrapQuotaRuleStateFreeClient(delegate QuotaRuleServiceClient) QuotaRuleServiceClient {
	return quotaRuleStateFreeClient{delegate: delegate}
}

func (c quotaRuleStateFreeClient) CreateOrUpdate(ctx context.Context, resource *filestoragev1beta1.QuotaRule, req ctrl.Request) (servicemanager.OSOKResponse, error) {
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err == nil && response.IsSuccessful && response.ShouldRequeue && resource != nil && resource.Status.OsokStatus.Ocid != "" {
		now := metav1.Now()
		resource.Status.OsokStatus.Message = "OCI quota rule is active"
		resource.Status.OsokStatus.Reason = string(shared.Active)
		resource.Status.OsokStatus.UpdatedAt = &now
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, corev1.ConditionTrue, "", resource.Status.OsokStatus.Message, loggerutil.OSOKLogger{})
		response.ShouldRequeue = false
		response.RequeueDuration = 0
	}
	return response, err
}

func (c quotaRuleStateFreeClient) Delete(ctx context.Context, resource *filestoragev1beta1.QuotaRule) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}
