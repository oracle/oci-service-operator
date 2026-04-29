/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package dkim

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	emailsdk "github.com/oracle/oci-go-sdk/v65/email"
	emailv1beta1 "github.com/oracle/oci-service-operator/api/email/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerDkimRuntimeHooksMutator(func(_ *DkimServiceManager, hooks *DkimRuntimeHooks) {
		applyDkimRuntimeHooks(hooks)
	})
}

func applyDkimRuntimeHooks(hooks *DkimRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = dkimRuntimeSemantics()
	hooks.BuildCreateBody = buildDkimCreateBody
	hooks.Identity.GuardExistingBeforeCreate = guardDkimExistingBeforeCreate
	hooks.List.Fields = dkimListFields()
	hooks.ParityHooks.NormalizeDesiredState = normalizeDkimDesiredState
}

func dkimRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "email",
		FormalSlug:    "dkim",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"DELETING"},
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"emailDomainId", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       []string{"definedTags", "description", "freeformTags"},
			ForceNew:      []string{"emailDomainId", "name"},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func dkimListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "EmailDomainId", RequestName: "emailDomainId", Contribution: "query"},
		{FieldName: "Id", RequestName: "id", Contribution: "query"},
		{FieldName: "Name", RequestName: "name", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func buildDkimCreateBody(_ context.Context, resource *emailv1beta1.Dkim, _ string) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("dkim resource is nil")
	}
	normalizeDkimSpec(resource)
	if resource.Spec.EmailDomainId == "" {
		return nil, fmt.Errorf("emailDomainId is required")
	}

	body := emailsdk.CreateDkimDetails{
		EmailDomainId: common.String(resource.Spec.EmailDomainId),
	}
	if resource.Spec.Name != "" {
		body.Name = common.String(resource.Spec.Name)
	}
	if resource.Spec.Description != "" {
		body.Description = common.String(resource.Spec.Description)
	}
	if len(resource.Spec.FreeformTags) > 0 {
		body.FreeformTags = cloneDkimStringMap(resource.Spec.FreeformTags)
	}
	if len(resource.Spec.DefinedTags) > 0 {
		body.DefinedTags = dkimDefinedTagsFromSpec(resource.Spec.DefinedTags)
	}
	return body, nil
}

func guardDkimExistingBeforeCreate(_ context.Context, resource *emailv1beta1.Dkim) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("dkim resource is nil")
	}
	normalizeDkimSpec(resource)
	if resource.Spec.EmailDomainId == "" || resource.Spec.Name == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func normalizeDkimDesiredState(resource *emailv1beta1.Dkim, _ any) {
	normalizeDkimSpec(resource)
}

func normalizeDkimSpec(resource *emailv1beta1.Dkim) {
	if resource == nil {
		return
	}
	resource.Spec.EmailDomainId = strings.TrimSpace(resource.Spec.EmailDomainId)
	resource.Spec.Name = strings.TrimSpace(resource.Spec.Name)
	resource.Spec.Description = strings.TrimSpace(resource.Spec.Description)
}

func cloneDkimStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func dkimDefinedTagsFromSpec(source map[string]shared.MapValue) map[string]map[string]interface{} {
	if len(source) == 0 {
		return nil
	}
	converted := make(map[string]map[string]interface{}, len(source))
	for namespace, values := range source {
		inner := make(map[string]interface{}, len(values))
		for key, value := range values {
			inner[key] = value
		}
		converted[namespace] = inner
	}
	return converted
}
