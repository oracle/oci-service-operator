/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package rovernode

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	roversdk "github.com/oracle/oci-go-sdk/v65/rover"
	roverv1beta1 "github.com/oracle/oci-service-operator/api/rover/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerRoverNodeRuntimeHooksMutator(func(_ *RoverNodeServiceManager, hooks *RoverNodeRuntimeHooks) {
		applyRoverNodeRuntimeHooks(hooks)
	})
}

func applyRoverNodeRuntimeHooks(hooks *RoverNodeRuntimeHooks) {
	if hooks == nil {
		return
	}

	hooks.Semantics = reviewedRoverNodeRuntimeSemantics()
	hooks.Identity.GuardExistingBeforeCreate = guardRoverNodeExistingBeforeCreate
	hooks.List.Fields = reviewedRoverNodeListFields()
	if hooks.List.Call != nil {
		hooks.List.Call = wrapRoverNodeListCallEnforceStandalone(hooks.List.Call)
	}
	hooks.ParityHooks.NormalizeDesiredState = normalizeRoverNodeDesiredState
	hooks.BuildUpdateBody = func(
		_ context.Context,
		resource *roverv1beta1.RoverNode,
		_ string,
		currentResponse any,
	) (any, bool, error) {
		return buildRoverNodeUpdateBody(resource, currentResponse)
	}
}

func reviewedRoverNodeRuntimeSemantics() *generatedruntime.Semantics {
	semantics := newRoverNodeRuntimeSemantics()
	semantics.List = &generatedruntime.ListSemantics{
		ResponseItemsField: "Items",
		MatchFields:        []string{"compartmentId", "displayName", "serialNumber", "shape"},
	}
	semantics.AuxiliaryOperations = nil
	return semantics
}

func reviewedRoverNodeListFields() []generatedruntime.RequestField {
	return []generatedruntime.RequestField{
		{FieldName: "CompartmentId", RequestName: "compartmentId", Contribution: "query"},
		{FieldName: "DisplayName", RequestName: "displayName", Contribution: "query"},
		{FieldName: "Shape", RequestName: "shape", Contribution: "query"},
		{FieldName: "Limit", RequestName: "limit", Contribution: "query"},
		{FieldName: "Page", RequestName: "page", Contribution: "query"},
		{FieldName: "SortOrder", RequestName: "sortOrder", Contribution: "query"},
		{FieldName: "SortBy", RequestName: "sortBy", Contribution: "query"},
	}
}

func guardRoverNodeExistingBeforeCreate(
	_ context.Context,
	resource *roverv1beta1.RoverNode,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil {
		return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
	}
	if strings.TrimSpace(resource.Spec.SerialNumber) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func wrapRoverNodeListCallEnforceStandalone(
	call func(context.Context, roversdk.ListRoverNodesRequest) (roversdk.ListRoverNodesResponse, error),
) func(context.Context, roversdk.ListRoverNodesRequest) (roversdk.ListRoverNodesResponse, error) {
	if call == nil {
		return nil
	}

	return func(ctx context.Context, request roversdk.ListRoverNodesRequest) (roversdk.ListRoverNodesResponse, error) {
		request.NodeType = roversdk.ListRoverNodesNodeTypeStandalone
		return call(ctx, request)
	}
}

func normalizeRoverNodeDesiredState(resource *roverv1beta1.RoverNode, _ any) {
	if resource == nil {
		return
	}

	resource.Spec.LifecycleState = ""
	resource.Spec.LifecycleStateDetails = ""
	resource.Spec.SystemTags = nil
	for i := range resource.Spec.NodeWorkloads {
		resource.Spec.NodeWorkloads[i].WorkRequestId = ""
	}
}

func buildRoverNodeUpdateBody(
	resource *roverv1beta1.RoverNode,
	currentResponse any,
) (roversdk.UpdateRoverNodeDetails, bool, error) {
	if resource == nil {
		return roversdk.UpdateRoverNodeDetails{}, false, fmt.Errorf("rovernode resource is nil")
	}

	current, err := roverNodeRuntimeBody(currentResponse)
	if err != nil {
		return roversdk.UpdateRoverNodeDetails{}, false, err
	}

	details := roversdk.UpdateRoverNodeDetails{}
	updateNeeded := false

	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.DisplayName, current.DisplayName); ok {
		details.DisplayName = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.Shape, current.Shape); ok {
		details.Shape = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.SerialNumber, current.SerialNumber); ok {
		details.SerialNumber = desired
		updateNeeded = true
	}
	if desired, ok, err := roverNodeDesiredShippingAddressUpdate(resource.Spec.CustomerShippingAddress, current.CustomerShippingAddress); err != nil {
		return roversdk.UpdateRoverNodeDetails{}, false, err
	} else if ok {
		details.CustomerShippingAddress = desired
		updateNeeded = true
	}
	if desired, ok, err := roverNodeDesiredWorkloadsUpdate(resource.Spec.NodeWorkloads, current.NodeWorkloads); err != nil {
		return roversdk.UpdateRoverNodeDetails{}, false, err
	} else if ok {
		details.NodeWorkloads = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.SuperUserPassword, current.SuperUserPassword); ok {
		details.SuperUserPassword = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.UnlockPassphrase, current.UnlockPassphrase); ok {
		details.UnlockPassphrase = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.PointOfContact, current.PointOfContact); ok {
		details.PointOfContact = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.PointOfContactPhoneNumber, current.PointOfContactPhoneNumber); ok {
		details.PointOfContactPhoneNumber = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.OracleShippingTrackingUrl, current.OracleShippingTrackingUrl); ok {
		details.OracleShippingTrackingUrl = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredShippingPreferenceUpdate(resource.Spec.ShippingPreference, current.ShippingPreference); ok {
		details.ShippingPreference = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.ShippingVendor, current.ShippingVendor); ok {
		details.ShippingVendor = desired
		updateNeeded = true
	}
	if desired, ok, err := roverNodeDesiredTimeUpdate("timePickupExpected", resource.Spec.TimePickupExpected, current.TimePickupExpected); err != nil {
		return roversdk.UpdateRoverNodeDetails{}, false, err
	} else if ok {
		details.TimePickupExpected = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.PublicKey, current.PublicKey); ok {
		details.PublicKey = desired
		updateNeeded = true
	}
	if desired, ok, err := roverNodeDesiredTimeUpdate("timeReturnWindowStarts", resource.Spec.TimeReturnWindowStarts, current.TimeReturnWindowStarts); err != nil {
		return roversdk.UpdateRoverNodeDetails{}, false, err
	} else if ok {
		details.TimeReturnWindowStarts = desired
		updateNeeded = true
	}
	if desired, ok, err := roverNodeDesiredTimeUpdate("timeReturnWindowEnds", resource.Spec.TimeReturnWindowEnds, current.TimeReturnWindowEnds); err != nil {
		return roversdk.UpdateRoverNodeDetails{}, false, err
	} else if ok {
		details.TimeReturnWindowEnds = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredEnclosureTypeUpdate(resource.Spec.EnclosureType, current.EnclosureType); ok {
		details.EnclosureType = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredBoolUpdate(resource.Spec.IsImportRequested, current.IsImportRequested); ok {
		details.IsImportRequested = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.ImportCompartmentId, current.ImportCompartmentId); ok {
		details.ImportCompartmentId = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.ImportFileBucket, current.ImportFileBucket); ok {
		details.ImportFileBucket = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.DataValidationCode, current.DataValidationCode); ok {
		details.DataValidationCode = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.CertificateAuthorityId, current.CertificateAuthorityId); ok {
		details.CertificateAuthorityId = desired
		updateNeeded = true
	}
	if desired, ok, err := roverNodeDesiredTimeUpdate("timeCertValidityEnd", resource.Spec.TimeCertValidityEnd, current.TimeCertValidityEnd); err != nil {
		return roversdk.UpdateRoverNodeDetails{}, false, err
	} else if ok {
		details.TimeCertValidityEnd = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.CommonName, current.CommonName); ok {
		details.CommonName = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredStringUpdate(resource.Spec.CertCompartmentId, current.CertCompartmentId); ok {
		details.CertCompartmentId = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredCertKeyAlgorithmUpdate(resource.Spec.CertKeyAlgorithm, current.CertKeyAlgorithm); ok {
		details.CertKeyAlgorithm = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredCertSignatureAlgorithmUpdate(resource.Spec.CertSignatureAlgorithm, current.CertSignatureAlgorithm); ok {
		details.CertSignatureAlgorithm = desired
		updateNeeded = true
	}
	if desired, ok := roverNodeDesiredFreeformTagsUpdate(resource.Spec.FreeformTags, current.FreeformTags); ok {
		details.FreeformTags = desired
		updateNeeded = true
	}
	if desired, ok, err := roverNodeDesiredDefinedTagsUpdate(resource.Spec.DefinedTags, current.DefinedTags); err != nil {
		return roversdk.UpdateRoverNodeDetails{}, false, err
	} else if ok {
		details.DefinedTags = desired
		updateNeeded = true
	}

	return details, updateNeeded, nil
}

func roverNodeRuntimeBody(currentResponse any) (roversdk.RoverNode, error) {
	switch current := currentResponse.(type) {
	case roversdk.RoverNode:
		return current, nil
	case *roversdk.RoverNode:
		if current == nil {
			return roversdk.RoverNode{}, fmt.Errorf("current RoverNode response is nil")
		}
		return *current, nil
	case roversdk.RoverNodeSummary:
		return roverNodeFromSummary(current), nil
	case *roversdk.RoverNodeSummary:
		if current == nil {
			return roversdk.RoverNode{}, fmt.Errorf("current RoverNode response is nil")
		}
		return roverNodeFromSummary(*current), nil
	case roversdk.CreateRoverNodeResponse:
		return current.RoverNode, nil
	case *roversdk.CreateRoverNodeResponse:
		if current == nil {
			return roversdk.RoverNode{}, fmt.Errorf("current RoverNode response is nil")
		}
		return current.RoverNode, nil
	case roversdk.GetRoverNodeResponse:
		return current.RoverNode, nil
	case *roversdk.GetRoverNodeResponse:
		if current == nil {
			return roversdk.RoverNode{}, fmt.Errorf("current RoverNode response is nil")
		}
		return current.RoverNode, nil
	case roversdk.UpdateRoverNodeResponse:
		return current.RoverNode, nil
	case *roversdk.UpdateRoverNodeResponse:
		if current == nil {
			return roversdk.RoverNode{}, fmt.Errorf("current RoverNode response is nil")
		}
		return current.RoverNode, nil
	default:
		return roversdk.RoverNode{}, fmt.Errorf("unexpected current RoverNode response type %T", currentResponse)
	}
}

func roverNodeFromSummary(summary roversdk.RoverNodeSummary) roversdk.RoverNode {
	return roversdk.RoverNode{
		Id:                    summary.Id,
		CompartmentId:         summary.CompartmentId,
		LifecycleState:        summary.LifecycleState,
		ClusterId:             summary.ClusterId,
		SerialNumber:          summary.SerialNumber,
		NodeType:              summary.NodeType,
		Shape:                 summary.Shape,
		DisplayName:           summary.DisplayName,
		TimeCreated:           summary.TimeCreated,
		LifecycleStateDetails: summary.LifecycleStateDetails,
		FreeformTags:          summary.FreeformTags,
		DefinedTags:           summary.DefinedTags,
		SystemTags:            summary.SystemTags,
	}
}

func roverNodeDesiredStringUpdate(spec string, current *string) (*string, bool) {
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

func roverNodeDesiredBoolUpdate(spec bool, current *bool) (*bool, bool) {
	currentValue := false
	if current != nil {
		currentValue = *current
	}
	if spec == currentValue {
		return nil, false
	}
	return common.Bool(spec), true
}

func roverNodeDesiredEnclosureTypeUpdate(
	spec string,
	current roversdk.EnclosureTypeEnum,
) (roversdk.EnclosureTypeEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return roversdk.EnclosureTypeEnum(spec), true
}

func roverNodeDesiredShippingPreferenceUpdate(
	spec string,
	current roversdk.RoverNodeShippingPreferenceEnum,
) (roversdk.UpdateRoverNodeDetailsShippingPreferenceEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return roversdk.UpdateRoverNodeDetailsShippingPreferenceEnum(spec), true
}

func roverNodeDesiredCertKeyAlgorithmUpdate(
	spec string,
	current roversdk.CertKeyAlgorithmEnum,
) (roversdk.CertKeyAlgorithmEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return roversdk.CertKeyAlgorithmEnum(spec), true
}

func roverNodeDesiredCertSignatureAlgorithmUpdate(
	spec string,
	current roversdk.CertSignatureAlgorithmEnum,
) (roversdk.CertSignatureAlgorithmEnum, bool) {
	if spec == "" || spec == string(current) {
		return "", false
	}
	return roversdk.CertSignatureAlgorithmEnum(spec), true
}

func roverNodeDesiredTimeUpdate(fieldName string, spec string, current *common.SDKTime) (*common.SDKTime, bool, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, false, nil
	}

	parsed, err := time.Parse(time.RFC3339, spec)
	if err != nil {
		return nil, false, fmt.Errorf("parse %s: %w", fieldName, err)
	}
	parsed = parsed.UTC()
	if current != nil && current.Time.Equal(parsed) {
		return nil, false, nil
	}

	return &common.SDKTime{Time: parsed}, true, nil
}

func roverNodeDesiredFreeformTagsUpdate(
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

func roverNodeDesiredDefinedTagsUpdate(
	spec map[string]shared.MapValue,
	current map[string]map[string]interface{},
) (map[string]map[string]interface{}, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false, nil
	}

	desired, err := roverNodeDefinedTagsFromSpec(spec)
	if err != nil {
		return nil, false, err
	}
	if roverNodeJSONEqual(desired, current) {
		return nil, false, nil
	}
	return desired, true, nil
}

func roverNodeDesiredShippingAddressUpdate(
	spec roverv1beta1.RoverNodeCustomerShippingAddress,
	current *roversdk.ShippingAddress,
) (*roversdk.ShippingAddress, bool, error) {
	if spec == (roverv1beta1.RoverNodeCustomerShippingAddress{}) {
		return nil, false, nil
	}

	desired, err := roverNodeShippingAddressFromSpec(spec)
	if err != nil {
		return nil, false, err
	}
	if current != nil && roverNodeJSONEqual(desired, current) {
		return nil, false, nil
	}
	return desired, true, nil
}

func roverNodeDesiredWorkloadsUpdate(
	spec []roverv1beta1.RoverNodeNodeWorkload,
	current []roversdk.RoverWorkload,
) ([]roversdk.RoverWorkload, bool, error) {
	if spec == nil {
		return nil, false, nil
	}
	if len(spec) == 0 && len(current) == 0 {
		return nil, false, nil
	}

	desired, err := roverNodeWorkloadsFromSpec(spec)
	if err != nil {
		return nil, false, err
	}
	if roverNodeJSONEqual(desired, current) {
		return nil, false, nil
	}
	return desired, true, nil
}

func roverNodeShippingAddressFromSpec(
	spec roverv1beta1.RoverNodeCustomerShippingAddress,
) (*roversdk.ShippingAddress, error) {
	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal RoverNode customerShippingAddress: %w", err)
	}

	var address roversdk.ShippingAddress
	if err := json.Unmarshal(payload, &address); err != nil {
		return nil, fmt.Errorf("decode RoverNode customerShippingAddress: %w", err)
	}
	return &address, nil
}

func roverNodeWorkloadsFromSpec(
	spec []roverv1beta1.RoverNodeNodeWorkload,
) ([]roversdk.RoverWorkload, error) {
	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal RoverNode nodeWorkloads: %w", err)
	}

	var workloads []roversdk.RoverWorkload
	if err := json.Unmarshal(payload, &workloads); err != nil {
		return nil, fmt.Errorf("decode RoverNode nodeWorkloads: %w", err)
	}
	return workloads, nil
}

func roverNodeDefinedTagsFromSpec(
	spec map[string]shared.MapValue,
) (map[string]map[string]interface{}, error) {
	payload, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("marshal RoverNode definedTags: %w", err)
	}

	var tags map[string]map[string]interface{}
	if err := json.Unmarshal(payload, &tags); err != nil {
		return nil, fmt.Errorf("decode RoverNode definedTags: %w", err)
	}
	return tags, nil
}

func roverNodeJSONEqual(left any, right any) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	if leftErr != nil || rightErr != nil {
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
	return string(leftPayload) == string(rightPayload)
}
