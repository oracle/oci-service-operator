/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package suppression

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	emailsdk "github.com/oracle/oci-go-sdk/v65/email"
	emailv1beta1 "github.com/oracle/oci-service-operator/api/email/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	suppressionActiveMessage        = "OCI suppression is active"
	suppressionDeletePendingMessage = "OCI suppression delete is in progress"
	suppressionDeletePendingState   = "DELETE_ACCEPTED"
)

func init() {
	registerSuppressionRuntimeHooksMutator(func(_ *SuppressionServiceManager, hooks *SuppressionRuntimeHooks) {
		applySuppressionRuntimeHooks(hooks)
	})
}

func applySuppressionRuntimeHooks(hooks *SuppressionRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newSuppressionRuntimeSemantics()
	hooks.BuildCreateBody = buildSuppressionCreateBody
	hooks.Identity.GuardExistingBeforeCreate = guardSuppressionExistingBeforeCreate
	hooks.List.Fields = suppressionListFields()
	hooks.ParityHooks.NormalizeDesiredState = normalizeSuppressionDesiredState
	hooks.StatusHooks.MarkTerminating = markSuppressionTerminating
	hooks.DeleteHooks.ApplyOutcome = applySuppressionDeleteOutcome
	hooks.WrapGeneratedClient = append(hooks.WrapGeneratedClient, wrapSuppressionStateFreeClient)
}

func newSuppressionRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "email",
		FormalSlug:    "suppression",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "none",
			Runtime:              "generatedruntime",
			FormalClassification: "none",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{suppressionDeletePendingState},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "emailAddress"},
		},
		Mutation: generatedruntime.MutationSemantics{
			ForceNew:      []string{"compartmentId", "emailAddress"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
	}
}

func suppressionListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "EmailAddress", RequestName: "emailAddress", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
	}
}

func buildSuppressionCreateBody(_ context.Context, resource *emailv1beta1.Suppression, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("suppression resource is nil")
	}
	normalizeSuppressionSpec(resource)
	if strings.TrimSpace(resource.Spec.CompartmentId) == "" {
		return nil, fmt.Errorf("compartmentId is required")
	}
	if strings.TrimSpace(resource.Spec.EmailAddress) == "" {
		return nil, fmt.Errorf("emailAddress is required")
	}

	return emailsdk.CreateSuppressionDetails{
		CompartmentId: common.String(resource.Spec.CompartmentId),
		EmailAddress:  common.String(resource.Spec.EmailAddress),
	}, nil
}

func guardSuppressionExistingBeforeCreate(_ context.Context, resource *emailv1beta1.Suppression) (generatedruntime.ExistingBeforeCreateDecision, error) {
	normalizeSuppressionSpec(resource)
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func normalizeSuppressionDesiredState(resource *emailv1beta1.Suppression, _ any) {
	normalizeSuppressionSpec(resource)
}

func normalizeSuppressionSpec(resource *emailv1beta1.Suppression) {
	if resource == nil {
		return
	}
	resource.Spec.CompartmentId = strings.TrimSpace(resource.Spec.CompartmentId)
	resource.Spec.EmailAddress = normalizeSuppressionEmail(resource.Spec.EmailAddress)
}

func normalizeSuppressionEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

type suppressionStateFreeClient struct {
	delegate SuppressionServiceClient
}

func wrapSuppressionStateFreeClient(delegate SuppressionServiceClient) SuppressionServiceClient {
	return suppressionStateFreeClient{delegate: delegate}
}

func (c suppressionStateFreeClient) CreateOrUpdate(
	ctx context.Context,
	resource *emailv1beta1.Suppression,
	req ctrl.Request,
) (servicemanager.OSOKResponse, error) {
	normalizeSuppressionSpec(resource)
	response, err := c.delegate.CreateOrUpdate(ctx, resource, req)
	if err != nil {
		return response, err
	}
	if response.IsSuccessful && response.ShouldRequeue && suppressionHasTrackedID(resource) {
		markSuppressionActive(resource)
		response.ShouldRequeue = false
		response.RequeueDuration = 0
	}
	return response, nil
}

func (c suppressionStateFreeClient) Delete(ctx context.Context, resource *emailv1beta1.Suppression) (bool, error) {
	return c.delegate.Delete(ctx, resource)
}

func suppressionHasTrackedID(resource *emailv1beta1.Suppression) bool {
	if resource == nil {
		return false
	}
	return strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)) != "" ||
		strings.TrimSpace(resource.Status.Id) != ""
}

func markSuppressionActive(resource *emailv1beta1.Suppression) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.Async.Current = nil
	status.Message = suppressionActiveMessage
	status.Reason = string(shared.Active)
	status.UpdatedAt = &now
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Active,
		v1.ConditionTrue,
		"",
		suppressionActiveMessage,
		loggerutil.OSOKLogger{},
	)
}

func applySuppressionDeleteOutcome(
	resource *emailv1beta1.Suppression,
	response any,
	stage generatedruntime.DeleteConfirmStage,
) (generatedruntime.DeleteOutcome, error) {
	if response == nil {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAlreadyPending && !suppressionDeleteAlreadyPending(resource) {
		return generatedruntime.DeleteOutcome{}, nil
	}
	if stage == generatedruntime.DeleteConfirmStageAfterRequest ||
		stage == generatedruntime.DeleteConfirmStageAlreadyPending {
		markSuppressionTerminating(resource, response)
		return generatedruntime.DeleteOutcome{Handled: true, Deleted: false}, nil
	}
	return generatedruntime.DeleteOutcome{}, nil
}

func suppressionDeleteAlreadyPending(resource *emailv1beta1.Suppression) bool {
	if resource == nil {
		return false
	}
	current := resource.Status.OsokStatus.Async.Current
	return current != nil &&
		current.Phase == shared.OSOKAsyncPhaseDelete &&
		current.NormalizedClass == shared.OSOKAsyncClassPending
}

func markSuppressionTerminating(resource *emailv1beta1.Suppression, _ any) {
	if resource == nil {
		return
	}
	now := metav1.Now()
	status := &resource.Status.OsokStatus
	status.UpdatedAt = &now
	status.Message = suppressionDeletePendingMessage
	status.Reason = string(shared.Terminating)
	status.Async.Current = &shared.OSOKAsyncOperation{
		Source:          shared.OSOKAsyncSourceLifecycle,
		Phase:           shared.OSOKAsyncPhaseDelete,
		RawStatus:       suppressionDeletePendingState,
		NormalizedClass: shared.OSOKAsyncClassPending,
		Message:         suppressionDeletePendingMessage,
		UpdatedAt:       &now,
	}
	*status = util.UpdateOSOKStatusCondition(
		*status,
		shared.Terminating,
		v1.ConditionTrue,
		"",
		suppressionDeletePendingMessage,
		loggerutil.OSOKLogger{},
	)
}
