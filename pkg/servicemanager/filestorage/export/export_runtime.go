/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package export

import (
	"context"
	"fmt"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/common"
	filestoragesdk "github.com/oracle/oci-go-sdk/v65/filestorage"
	filestoragev1beta1 "github.com/oracle/oci-service-operator/api/filestorage/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

func init() {
	registerExportRuntimeHooksMutator(func(_ *ExportServiceManager, hooks *ExportRuntimeHooks) {
		if hooks != nil {
			hooks.Semantics = reviewedExportRuntimeSemantics()
			hooks.BuildUpdateBody = buildExportUpdateBody
		}
	})
}

func reviewedExportRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "filestorage", FormalSlug: "export", StatusProjection: "required",
		SecretSideEffects: "none", FinalizerPolicy: "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"CREATING"}, ActiveStates: []string{"ACTIVE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy: "required", PendingStates: []string{"DELETING"}, TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items", MatchFields: []string{"exportSetId", "fileSystemId", "path"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{"exportOptions", "isIdmapGroupsForSysAuth"},
			ForceNew: []string{"exportSetId", "fileSystemId", "path"},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

func buildExportUpdateBody(
	_ context.Context,
	resource *filestoragev1beta1.Export,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return filestoragesdk.UpdateExportDetails{}, false, fmt.Errorf("export resource is nil")
	}
	current, ok := exportFromResponse(currentResponse)
	if !ok {
		return filestoragesdk.UpdateExportDetails{}, false, fmt.Errorf("unexpected Export current response type %T", currentResponse)
	}
	desiredOptions := exportClientOptions(resource.Spec.ExportOptions)
	details := filestoragesdk.UpdateExportDetails{}
	needed := false
	if resource.Spec.ExportOptions != nil && !reflect.DeepEqual(current.ExportOptions, desiredOptions) {
		details.ExportOptions = desiredOptions
		needed = true
	}
	if resource.Spec.IsIdmapGroupsForSysAuth != exportBoolValue(current.IsIdmapGroupsForSysAuth) {
		details.IsIdmapGroupsForSysAuth = common.Bool(resource.Spec.IsIdmapGroupsForSysAuth)
		needed = true
	}
	return details, needed, nil
}

func exportBoolValue(value *bool) bool {
	return value != nil && *value
}

func exportFromResponse(response any) (filestoragesdk.Export, bool) {
	switch typed := response.(type) {
	case filestoragesdk.Export:
		return typed, true
	case *filestoragesdk.Export:
		if typed != nil {
			return *typed, true
		}
	case filestoragesdk.CreateExportResponse:
		return typed.Export, true
	case filestoragesdk.GetExportResponse:
		return typed.Export, true
	case filestoragesdk.UpdateExportResponse:
		return typed.Export, true
	}
	return filestoragesdk.Export{}, false
}

func exportClientOptions(options []filestoragev1beta1.ExportOption) []filestoragesdk.ClientOptions {
	if options == nil {
		return nil
	}
	result := make([]filestoragesdk.ClientOptions, 0, len(options))
	for _, option := range options {
		converted := filestoragesdk.ClientOptions{
			Source:                      common.String(option.Source),
			RequirePrivilegedSourcePort: common.Bool(option.RequirePrivilegedSourcePort),
			Access:                      filestoragesdk.ClientOptionsAccessEnum(option.Access),
			IdentitySquash:              filestoragesdk.ClientOptionsIdentitySquashEnum(option.IdentitySquash),
			IsAnonymousAccessAllowed:    common.Bool(option.IsAnonymousAccessAllowed),
		}
		if option.AnonymousUid != 0 {
			converted.AnonymousUid = common.Int64(option.AnonymousUid)
		}
		if option.AnonymousGid != 0 {
			converted.AnonymousGid = common.Int64(option.AnonymousGid)
		}
		for _, auth := range option.AllowedAuth {
			converted.AllowedAuth = append(converted.AllowedAuth, filestoragesdk.ClientOptionsAllowedAuthEnum(auth))
		}
		result = append(result, converted)
	}
	return result
}
