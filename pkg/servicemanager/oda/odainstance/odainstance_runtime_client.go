/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package odainstance

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	odasdk "github.com/oracle/oci-go-sdk/v65/oda"
	odav1beta1 "github.com/oracle/oci-service-operator/api/oda/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerOdaInstanceRuntimeHooksMutator(func(_ *OdaInstanceServiceManager, hooks *OdaInstanceRuntimeHooks) {
		applyOdaInstanceRuntimeHooks(hooks)
	})
}

func applyOdaInstanceRuntimeHooks(hooks *OdaInstanceRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = newOdaInstanceRuntimeSemantics()
	hooks.Identity = generatedruntime.IdentityHooks[*odav1beta1.OdaInstance]{
		GuardExistingBeforeCreate: guardOdaInstanceExistingBeforeCreate,
	}
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *odav1beta1.OdaInstance,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildOdaInstanceUpdateBody(resource, currentResponse)
	}
}

func newOdaInstanceRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService: "oda",
		FormalSlug:    "odainstance",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "lifecycle",
			Runtime:              "generatedruntime",
			FormalClassification: "lifecycle",
		},
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{string(odasdk.OdaInstanceLifecycleStateCreating)},
			UpdatingStates:     []string{string(odasdk.OdaInstanceLifecycleStateUpdating)},
			ActiveStates: []string{
				string(odasdk.OdaInstanceLifecycleStateActive),
				string(odasdk.OdaInstanceLifecycleStateInactive),
			},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{string(odasdk.OdaInstanceLifecycleStateDeleting)},
			TerminalStates: []string{string(odasdk.OdaInstanceLifecycleStateDeleted)},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"compartmentId", "displayName"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable: []string{
				"displayName",
				"description",
				"freeformTags",
				"definedTags",
			},
			ForceNew: []string{
				"compartmentId",
				"shapeName",
				"isRoleBasedAccess",
				"identityDomain",
			},
			ConflictsWith: map[string][]string{},
		},
		Hooks: generatedruntime.HookSet{
			Create: []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "OdaInstance", Action: "CreateOdaInstance"}},
			Update: []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "OdaInstance", Action: "UpdateOdaInstance"}},
			Delete: []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "OdaInstance", Action: "DeleteOdaInstance"}},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.CreateResource", EntityType: "OdaInstance", Action: "GetOdaInstance"}},
		},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "read-after-write",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.UpdateResource", EntityType: "OdaInstance", Action: "GetOdaInstance"}},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource", EntityType: "OdaInstance", Action: "GetOdaInstance"}},
		},
		AuxiliaryOperations: []generatedruntime.AuxiliaryOperation{},
		Unsupported:         []generatedruntime.UnsupportedSemantic{},
	}
}

func guardOdaInstanceExistingBeforeCreate(_ context.Context, resource *odav1beta1.OdaInstance) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("OdaInstance resource is nil")
	}
	if strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildOdaInstanceUpdateBody(
	resource *odav1beta1.OdaInstance,
	currentResponse any,
) (odasdk.UpdateOdaInstanceDetails, bool, error) {
	if resource == nil {
		return odasdk.UpdateOdaInstanceDetails{}, false, fmt.Errorf("OdaInstance resource is nil")
	}

	current, err := odaInstanceRuntimeBody(currentResponse)
	if err != nil {
		return odasdk.UpdateOdaInstanceDetails{}, false, err
	}

	details := odasdk.UpdateOdaInstanceDetails{}
	updateNeeded := false

	if desired, ok := odaInstanceDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := odaInstanceDesiredStringUpdate(resource.Spec.Description, current.Description); ok {
		details.Description = desired
		updateNeeded = true
	}
	if desired, ok := odaInstanceDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok := odaInstanceDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func odaInstanceRuntimeBody(currentResponse any) (odasdk.OdaInstance, error) {
	switch current := currentResponse.(type) {
	case odasdk.OdaInstance:
		return current, nil
	case *odasdk.OdaInstance:
		if current == nil {
			return odasdk.OdaInstance{}, fmt.Errorf("current OdaInstance response is nil")
		}
		return *current, nil
	case odasdk.OdaInstanceSummary:
		return odaInstanceFromSummary(current), nil
	case *odasdk.OdaInstanceSummary:
		if current == nil {
			return odasdk.OdaInstance{}, fmt.Errorf("current OdaInstance response is nil")
		}
		return odaInstanceFromSummary(*current), nil
	case odasdk.CreateOdaInstanceResponse:
		return current.OdaInstance, nil
	case *odasdk.CreateOdaInstanceResponse:
		if current == nil {
			return odasdk.OdaInstance{}, fmt.Errorf("current OdaInstance response is nil")
		}
		return current.OdaInstance, nil
	case odasdk.GetOdaInstanceResponse:
		return current.OdaInstance, nil
	case *odasdk.GetOdaInstanceResponse:
		if current == nil {
			return odasdk.OdaInstance{}, fmt.Errorf("current OdaInstance response is nil")
		}
		return current.OdaInstance, nil
	case odasdk.UpdateOdaInstanceResponse:
		return current.OdaInstance, nil
	case *odasdk.UpdateOdaInstanceResponse:
		if current == nil {
			return odasdk.OdaInstance{}, fmt.Errorf("current OdaInstance response is nil")
		}
		return current.OdaInstance, nil
	default:
		return odasdk.OdaInstance{}, fmt.Errorf("unexpected current OdaInstance response type %T", currentResponse)
	}
}

func odaInstanceFromSummary(summary odasdk.OdaInstanceSummary) odasdk.OdaInstance {
	return odasdk.OdaInstance{
		Id:                   summary.Id,
		CompartmentId:        summary.CompartmentId,
		ShapeName:            odasdk.OdaInstanceShapeNameEnum(summary.ShapeName),
		DisplayName:          summary.DisplayName,
		Description:          summary.Description,
		TimeCreated:          summary.TimeCreated,
		TimeUpdated:          summary.TimeUpdated,
		LifecycleState:       odasdk.OdaInstanceLifecycleStateEnum(summary.LifecycleState),
		LifecycleSubState:    odasdk.OdaInstanceLifecycleSubStateEnum(summary.LifecycleSubState),
		StateMessage:         summary.StateMessage,
		FreeformTags:         summary.FreeformTags,
		DefinedTags:          summary.DefinedTags,
		IsRoleBasedAccess:    summary.IsRoleBasedAccess,
		IdentityDomain:       summary.IdentityDomain,
		ImportedPackageNames: append([]string(nil), summary.ImportedPackageNames...),
		AttachmentTypes:      append([]string(nil), summary.AttachmentTypes...),
	}
}

func odaInstanceDesiredStringUpdate(spec string, current *string) (*string, bool) {
	currentValue := ""
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	if spec == "" && current == nil {
		return nil, false
	}
	return common.String(spec), true
}

func odaInstanceDesiredFreeformTagsUpdate(
	spec map[string]string,
	current map[string]string,
) (map[string]string, bool) {
	if spec == nil {
		return nil, false
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false
	}
	if maps.Equal(spec, current) {
		return nil, false
	}
	return maps.Clone(spec), true
}

func odaInstanceDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool) {
	if spec == nil {
		return nil, false
	}

	desired := odaInstanceDefinedTagsFromSpec(spec)
	if len(desired) == 0 && len(current) == 0 {
		return nil, false
	}
	if odaInstanceJSONEqual(desired, current) {
		return nil, false
	}
	return desired, true
}

func odaInstanceDefinedTagsFromSpec(spec map[string]shared.MapValue) map[string]map[string]interface{} {
	if spec == nil {
		return nil
	}

	desired := make(map[string]map[string]interface{}, len(spec))
	for namespace, values := range spec {
		converted := make(map[string]interface{}, len(values))
		for key, value := range values {
			converted[key] = value
		}
		desired[namespace] = converted
	}
	return desired
}

func odaInstanceJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftPayload) == string(rightPayload)
}
