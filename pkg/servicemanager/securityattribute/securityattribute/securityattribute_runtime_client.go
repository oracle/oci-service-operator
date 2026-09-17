/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package securityattribute

import (
	"context"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	securityattributesdk "github.com/oracle/oci-go-sdk/v65/securityattribute"
	securityattributev1beta1 "github.com/oracle/oci-service-operator/api/securityattribute/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

type securityAttributeIdentity struct {
	namespaceID string
	name        string
}

func init() {
	registerSecurityAttributeRuntimeHooksMutator(func(_ *SecurityAttributeServiceManager, hooks *SecurityAttributeRuntimeHooks) {
		applySecurityAttributeRuntimeHooks(hooks)
	})
}

func applySecurityAttributeRuntimeHooks(hooks *SecurityAttributeRuntimeHooks) {
	if hooks == nil {
		return
	}
	hooks.Semantics = securityAttributeRuntimeSemantics()
	get := hooks.Get.Call
	hooks.Identity = generatedruntime.IdentityHooks[*securityattributev1beta1.SecurityAttribute]{
		RecordBeforeCreateFollowUp: true,
		Resolve: func(resource *securityattributev1beta1.SecurityAttribute) (any, error) {
			return resolveSecurityAttributeIdentity(resource)
		},
		RecordPath: func(resource *securityattributev1beta1.SecurityAttribute, identity any) {
			recordSecurityAttributePath(resource, identity.(securityAttributeIdentity))
		},
		RecordTracked: func(resource *securityattributev1beta1.SecurityAttribute, identity any, resourceID string) {
			recordSecurityAttributeTracked(resource, identity.(securityAttributeIdentity), resourceID)
		},
		LookupExisting: func(ctx context.Context, _ *securityattributev1beta1.SecurityAttribute, identity any) (any, error) {
			if get == nil {
				return nil, nil
			}
			resolved := identity.(securityAttributeIdentity)
			return get(ctx, securityattributesdk.GetSecurityAttributeRequest{SecurityAttributeNamespaceId: common.String(resolved.namespaceID), SecurityAttributeName: common.String(resolved.name)})
		},
	}
	hooks.Create.Fields = []generatedruntime.RequestField{securityAttributeNamespaceIDField(), {FieldName: "CreateSecurityAttributeDetails", Contribution: "body"}}
	hooks.Get.Fields = []generatedruntime.RequestField{securityAttributeNamespaceIDField(), securityAttributeNameField()}
	hooks.List.Fields = []generatedruntime.RequestField{securityAttributeNamespaceIDField()}
	hooks.Update.Fields = []generatedruntime.RequestField{securityAttributeNamespaceIDField(), securityAttributeNameField(), {FieldName: "UpdateSecurityAttributeDetails", Contribution: "body"}}
	hooks.Delete.Fields = []generatedruntime.RequestField{securityAttributeNamespaceIDField(), securityAttributeNameField()}
}

func securityAttributeRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		Async:             &generatedruntime.AsyncSemantics{Strategy: "lifecycle", Runtime: "generatedruntime", FormalClassification: "lifecycle"},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle:         generatedruntime.LifecycleSemantics{ActiveStates: []string{"ACTIVE", "INACTIVE"}},
		Delete:            generatedruntime.DeleteSemantics{Policy: "required", TerminalStates: []string{"DELETED"}},
		List:              &generatedruntime.ListSemantics{ResponseItemsField: "Items", MatchFields: []string{"name"}},
		Mutation:          generatedruntime.MutationSemantics{Mutable: []string{"description", "isRetired", "validator"}, ForceNew: []string{"name", "securityAttributeNamespaceId"}, ConflictsWith: map[string][]string{}},
		CreateFollowUp:    generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp:    generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp:    generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func securityAttributeNamespaceIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{FieldName: "SecurityAttributeNamespaceId", RequestName: "securityAttributeNamespaceId", Contribution: "path", LookupPaths: []string{"status.securityAttributeNamespaceId", "spec.securityAttributeNamespaceId"}}
}

func securityAttributeNameField() generatedruntime.RequestField {
	return generatedruntime.RequestField{FieldName: "SecurityAttributeName", RequestName: "securityAttributeName", Contribution: "path", LookupPaths: []string{"status.name", "spec.name", "name"}}
}

func resolveSecurityAttributeIdentity(resource *securityattributev1beta1.SecurityAttribute) (securityAttributeIdentity, error) {
	if resource == nil {
		return securityAttributeIdentity{}, fmt.Errorf("resolve SecurityAttribute identity: resource is nil")
	}
	statusNamespaceID := strings.TrimSpace(resource.Status.SecurityAttributeNamespaceId)
	specNamespaceID := strings.TrimSpace(resource.Spec.SecurityAttributeNamespaceId)
	if statusNamespaceID != "" && specNamespaceID != "" && statusNamespaceID != specNamespaceID {
		return securityAttributeIdentity{}, fmt.Errorf("resolve SecurityAttribute identity: securityAttributeNamespaceId changed from %q to %q", statusNamespaceID, specNamespaceID)
	}
	statusName := strings.TrimSpace(resource.Status.Name)
	specName := strings.TrimSpace(resource.Spec.Name)
	if statusName != "" && specName != "" && statusName != specName {
		return securityAttributeIdentity{}, fmt.Errorf("resolve SecurityAttribute identity: name changed from %q to %q", statusName, specName)
	}
	identity := securityAttributeIdentity{namespaceID: firstSecurityAttributeValue(statusNamespaceID, specNamespaceID), name: firstSecurityAttributeValue(statusName, specName, resource.Name)}
	if identity.namespaceID == "" {
		return securityAttributeIdentity{}, fmt.Errorf("resolve SecurityAttribute identity: spec.securityAttributeNamespaceId is required")
	}
	if identity.name == "" {
		return securityAttributeIdentity{}, fmt.Errorf("resolve SecurityAttribute identity: spec.name is required")
	}
	return identity, nil
}

func recordSecurityAttributePath(resource *securityattributev1beta1.SecurityAttribute, identity securityAttributeIdentity) {
	if resource == nil {
		return
	}
	resource.Status.SecurityAttributeNamespaceId = identity.namespaceID
	resource.Status.Name = identity.name
}

func recordSecurityAttributeTracked(resource *securityattributev1beta1.SecurityAttribute, identity securityAttributeIdentity, resourceID string) {
	recordSecurityAttributePath(resource, identity)
	if resourceID == "" {
		resourceID = strings.TrimSpace(resource.Status.Id)
	}
	if resourceID != "" {
		resource.Status.OsokStatus.Ocid = shared.OCID(resourceID)
	}
}

func firstSecurityAttributeValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
