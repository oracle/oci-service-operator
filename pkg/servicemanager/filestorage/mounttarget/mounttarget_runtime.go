/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package mounttarget

import (
	"context"
	"fmt"
	"reflect"

	"github.com/oracle/oci-go-sdk/v65/common"
	filestoragesdk "github.com/oracle/oci-go-sdk/v65/filestorage"
	filestoragev1beta1 "github.com/oracle/oci-service-operator/api/filestorage/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/util"
)

func init() {
	registerMountTargetRuntimeHooksMutator(func(_ *MountTargetServiceManager, hooks *MountTargetRuntimeHooks) {
		if hooks != nil {
			hooks.Semantics = reviewedMountTargetRuntimeSemantics()
			hooks.BuildUpdateBody = buildMountTargetUpdateBody
		}
	})
}

func buildMountTargetUpdateBody(
	_ context.Context,
	resource *filestoragev1beta1.MountTarget,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return filestoragesdk.UpdateMountTargetDetails{}, false, fmt.Errorf("mount target resource is nil")
	}
	current, ok := mountTargetFromResponse(currentResponse)
	if !ok {
		return filestoragesdk.UpdateMountTargetDetails{}, false, fmt.Errorf(
			"unexpected MountTarget current response type %T",
			currentResponse,
		)
	}

	details := filestoragesdk.UpdateMountTargetDetails{}
	updateNeeded := false
	if resource.Spec.DisplayName != "" &&
		(current.DisplayName == nil || *current.DisplayName != resource.Spec.DisplayName) {
		details.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.NsgIds != nil && !reflect.DeepEqual(current.NsgIds, resource.Spec.NsgIds) {
		details.NsgIds = append([]string(nil), resource.Spec.NsgIds...)
		updateNeeded = true
	}
	if resource.Spec.FreeformTags != nil &&
		!reflect.DeepEqual(current.FreeformTags, resource.Spec.FreeformTags) {
		details.FreeformTags = cloneMountTargetStringMap(resource.Spec.FreeformTags)
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil {
		desired := *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desired) {
			details.DefinedTags = desired
			updateNeeded = true
		}
	}
	if resource.Spec.SecurityAttributes != nil {
		desired := *util.ConvertToOciDefinedTags(&resource.Spec.SecurityAttributes)
		if !reflect.DeepEqual(current.SecurityAttributes, desired) {
			details.SecurityAttributes = desired
			updateNeeded = true
		}
	}
	return details, updateNeeded, nil
}

func mountTargetFromResponse(response any) (filestoragesdk.MountTarget, bool) {
	switch typed := response.(type) {
	case filestoragesdk.MountTarget:
		return typed, true
	case *filestoragesdk.MountTarget:
		if typed != nil {
			return *typed, true
		}
	case filestoragesdk.CreateMountTargetResponse:
		return typed.MountTarget, true
	case filestoragesdk.GetMountTargetResponse:
		return typed.MountTarget, true
	case filestoragesdk.UpdateMountTargetResponse:
		return typed.MountTarget, true
	}
	return filestoragesdk.MountTarget{}, false
}

func cloneMountTargetStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func reviewedMountTargetRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "filestorage",
		FormalSlug:        "mounttarget",
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
			MatchFields:        []string{"availabilityDomain", "compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"definedTags",
				"displayName",
				"freeformTags",
				"nsgIds",
				"securityAttributes",
			},
			ForceNew: []string{
				"availabilityDomain",
				"compartmentId",
				"hostnameLabel",
				"ipAddress",
				"subnetId",
			},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}
