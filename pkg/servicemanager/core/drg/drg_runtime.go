/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package drg

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	coresdk "github.com/oracle/oci-go-sdk/v65/core"
	corev1beta1 "github.com/oracle/oci-service-operator/api/core/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
	"github.com/oracle/oci-service-operator/pkg/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const drgRequeueDuration = time.Minute

func init() {
	registerDrgRuntimeHooksMutator(func(manager *DrgServiceManager, hooks *DrgRuntimeHooks) {
		applyDrgRuntimeHooks(manager, hooks)
	})
}

func applyDrgRuntimeHooks(manager *DrgServiceManager, hooks *DrgRuntimeHooks) {
	if hooks == nil {
		return
	}

	runtime := drgRuntimeHooks{manager: manager}

	hooks.Semantics = newDrgRuntimeSemantics()
	hooks.BuildCreateBody = func(_ context.Context, resource *corev1beta1.Drg, _ string) (any, error) {
		return buildCreateDrgDetails(resource.Spec), nil
	}
	hooks.BuildUpdateBody = func(_ context.Context, resource *corev1beta1.Drg, _ string, currentResponse any) (any, bool, error) {
		current, ok := drgFromResponse(currentResponse)
		if !ok {
			return nil, false, fmt.Errorf("unexpected Drg current response type %T", currentResponse)
		}
		request, updateNeeded, err := buildUpdateDrgRequest(resource, current)
		if err != nil {
			return nil, false, err
		}
		return request.UpdateDrgDetails, updateNeeded, nil
	}
	hooks.Identity.GuardExistingBeforeCreate = guardDrgExistingBeforeCreate
	hooks.TrackedRecreate.ClearTrackedIdentity = clearTrackedDrgIdentity
	hooks.StatusHooks.ClearProjectedStatus = clearDrgProjectedStatus
	hooks.StatusHooks.RestoreStatus = restoreDrgStatus
	hooks.StatusHooks.ProjectStatus = projectDrgStatus
	hooks.StatusHooks.ApplyLifecycle = runtime.applyLifecycle
	hooks.StatusHooks.MarkDeleted = runtime.markDeleted
	hooks.StatusHooks.MarkTerminating = runtime.markTerminating
	hooks.ParityHooks.ValidateCreateOnlyDrift = func(resource *corev1beta1.Drg, response any) error {
		current, ok := drgFromResponse(response)
		if !ok {
			return fmt.Errorf("unexpected Drg current response type %T", response)
		}
		return validateDrgCreateOnlyDrift(resource.Spec, current)
	}
}

type drgRuntimeHooks struct {
	manager *DrgServiceManager
}

func newDrgRuntimeSemantics() *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "core",
		FormalSlug:        "drg",
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Lifecycle: generatedruntime.LifecycleSemantics{
			ProvisioningStates: []string{"PROVISIONING"},
			UpdatingStates:     []string{"UPDATING"},
			ActiveStates:       []string{"AVAILABLE"},
		},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			PendingStates:  []string{"TERMINATING"},
			TerminalStates: []string{"TERMINATED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"displayName", "compartmentId"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:  []string{"displayName", "definedTags", "freeformTags", "defaultDrgRouteTables"},
			ForceNew: []string{"compartmentId"},
		},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{
			Strategy: "confirm-delete",
			Hooks:    []generatedruntime.Hook{{Helper: "tfresource.DeleteResource"}},
		},
	}
}

func guardDrgExistingBeforeCreate(_ context.Context, resource *corev1beta1.Drg) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if resource == nil || strings.TrimSpace(resource.Spec.DisplayName) == "" {
		return generatedruntime.ExistingBeforeCreateDecisionSkip, nil
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func buildCreateDrgDetails(spec corev1beta1.DrgSpec) coresdk.CreateDrgDetails {
	createDetails := coresdk.CreateDrgDetails{
		CompartmentId: common.String(spec.CompartmentId),
	}
	if spec.DefinedTags != nil {
		createDetails.DefinedTags = *util.ConvertToOciDefinedTags(&spec.DefinedTags)
	}
	if spec.DisplayName != "" {
		createDetails.DisplayName = common.String(spec.DisplayName)
	}
	if spec.FreeformTags != nil {
		createDetails.FreeformTags = spec.FreeformTags
	}
	return createDetails
}

func buildUpdateDrgRequest(resource *corev1beta1.Drg, current coresdk.Drg) (coresdk.UpdateDrgRequest, bool, error) {
	if current.Id == nil || strings.TrimSpace(*current.Id) == "" {
		return coresdk.UpdateDrgRequest{}, false, fmt.Errorf("current Drg does not expose an OCI identifier")
	}
	if err := validateDrgCreateOnlyDrift(resource.Spec, current); err != nil {
		return coresdk.UpdateDrgRequest{}, false, err
	}

	updateDetails := coresdk.UpdateDrgDetails{}
	updateNeeded := false

	if resource.Spec.DisplayName != "" && !stringPtrEqual(current.DisplayName, resource.Spec.DisplayName) {
		updateDetails.DisplayName = common.String(resource.Spec.DisplayName)
		updateNeeded = true
	}
	if resource.Spec.FreeformTags != nil && !reflect.DeepEqual(current.FreeformTags, resource.Spec.FreeformTags) {
		updateDetails.FreeformTags = resource.Spec.FreeformTags
		updateNeeded = true
	}
	if resource.Spec.DefinedTags != nil {
		desiredDefinedTags := *util.ConvertToOciDefinedTags(&resource.Spec.DefinedTags)
		if !reflect.DeepEqual(current.DefinedTags, desiredDefinedTags) {
			updateDetails.DefinedTags = desiredDefinedTags
			updateNeeded = true
		}
	}
	if defaultDrgRouteTablesSpecified(resource.Spec.DefaultDrgRouteTables) &&
		!defaultDrgRouteTablesMatch(current.DefaultDrgRouteTables, resource.Spec.DefaultDrgRouteTables) {
		updateDetails.DefaultDrgRouteTables = buildDefaultDrgRouteTables(resource.Spec.DefaultDrgRouteTables)
		updateNeeded = true
	}

	if !updateNeeded {
		return coresdk.UpdateDrgRequest{}, false, nil
	}
	return coresdk.UpdateDrgRequest{
		DrgId:            current.Id,
		UpdateDrgDetails: updateDetails,
	}, true, nil
}

func validateDrgCreateOnlyDrift(spec corev1beta1.DrgSpec, current coresdk.Drg) error {
	if stringCreateOnlyMatches(current.CompartmentId, spec.CompartmentId) {
		return nil
	}
	return fmt.Errorf("Drg create-only field drift is not supported: compartmentId")
}

func (h drgRuntimeHooks) applyLifecycle(resource *corev1beta1.Drg, response any) (servicemanager.OSOKResponse, error) {
	current, ok := drgFromResponse(response)
	if !ok {
		return h.fail(resource, fmt.Errorf("unexpected Drg lifecycle response type %T", response))
	}

	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	if status.CreatedAt == nil && strings.TrimSpace(stringValue(current.Id)) != "" {
		status.CreatedAt = &now
	}
	status.UpdatedAt = &now
	if current.Id != nil {
		status.Ocid = shared.OCID(*current.Id)
	}

	message := drgLifecycleMessage(current)
	status.Message = message

	switch strings.ToUpper(string(current.LifecycleState)) {
	case "AVAILABLE":
		status.Reason = string(shared.Active)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Active, v1.ConditionTrue, "", message, h.log())
		return servicemanager.OSOKResponse{IsSuccessful: true}, nil
	case "PROVISIONING":
		status.Reason = string(shared.Provisioning)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Provisioning, v1.ConditionTrue, "", message, h.log())
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: drgRequeueDuration}, nil
	case "UPDATING":
		status.Reason = string(shared.Updating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Updating, v1.ConditionTrue, "", message, h.log())
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: drgRequeueDuration}, nil
	case "TERMINATING", "TERMINATED":
		status.Reason = string(shared.Terminating)
		resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, h.log())
		return servicemanager.OSOKResponse{IsSuccessful: true, ShouldRequeue: true, RequeueDuration: drgRequeueDuration}, nil
	default:
		return h.fail(resource, fmt.Errorf("Drg lifecycle state %q is not modeled for create or update", current.LifecycleState))
	}
}

func (h drgRuntimeHooks) fail(resource *corev1beta1.Drg, err error) (servicemanager.OSOKResponse, error) {
	status := &resource.Status.OsokStatus
	status.Message = err.Error()
	status.Reason = string(shared.Failed)
	updatedAt := metav1Time(time.Now())
	status.UpdatedAt = &updatedAt
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Failed, v1.ConditionFalse, "", err.Error(), h.log())
	return servicemanager.OSOKResponse{IsSuccessful: false}, err
}

func (h drgRuntimeHooks) markDeleted(resource *corev1beta1.Drg, message string) {
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.DeletedAt = &now
	status.UpdatedAt = &now
	status.Message = message
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", message, h.log())
}

func (h drgRuntimeHooks) markTerminating(resource *corev1beta1.Drg, response any) {
	current, ok := drgFromResponse(response)
	if !ok {
		return
	}
	status := &resource.Status.OsokStatus
	now := metav1Time(time.Now())
	status.UpdatedAt = &now
	status.Message = drgLifecycleMessage(current)
	status.Reason = string(shared.Terminating)
	resource.Status.OsokStatus = util.UpdateOSOKStatusCondition(resource.Status.OsokStatus, shared.Terminating, v1.ConditionTrue, "", status.Message, h.log())
}

func (h drgRuntimeHooks) log() loggerutil.OSOKLogger {
	if h.manager == nil {
		return loggerutil.OSOKLogger{}
	}
	return h.manager.Log
}

func projectDrgStatus(resource *corev1beta1.Drg, response any) error {
	current, ok := drgFromResponse(response)
	if !ok {
		return fmt.Errorf("unexpected Drg status response type %T", response)
	}
	resource.Status = corev1beta1.DrgStatus{
		OsokStatus:                          resource.Status.OsokStatus,
		CompartmentId:                       stringValue(current.CompartmentId),
		Id:                                  stringValue(current.Id),
		LifecycleState:                      string(current.LifecycleState),
		DefinedTags:                         convertOCIToStatusDefinedTags(current.DefinedTags),
		DisplayName:                         stringValue(current.DisplayName),
		FreeformTags:                        cloneStringMap(current.FreeformTags),
		TimeCreated:                         sdkTimeString(current.TimeCreated),
		DefaultDrgRouteTables:               statusDefaultDrgRouteTables(current.DefaultDrgRouteTables),
		DefaultExportDrgRouteDistributionId: stringValue(current.DefaultExportDrgRouteDistributionId),
	}
	return nil
}

func clearDrgProjectedStatus(resource *corev1beta1.Drg) any {
	if resource == nil {
		return corev1beta1.DrgStatus{}
	}
	previous := resource.Status
	resource.Status = corev1beta1.DrgStatus{
		OsokStatus: previous.OsokStatus,
		Id:         previous.Id,
	}
	return previous
}

func restoreDrgStatus(resource *corev1beta1.Drg, baseline any) {
	if resource == nil {
		return
	}
	previous, ok := baseline.(corev1beta1.DrgStatus)
	if !ok {
		return
	}
	failedStatus := resource.Status.OsokStatus
	resource.Status = previous
	resource.Status.OsokStatus = failedStatus
}

func clearTrackedDrgIdentity(resource *corev1beta1.Drg) {
	if resource == nil {
		return
	}
	resource.Status = corev1beta1.DrgStatus{}
}

func drgFromResponse(response any) (coresdk.Drg, bool) {
	switch typed := response.(type) {
	case coresdk.Drg:
		return typed, true
	case coresdk.CreateDrgResponse:
		return typed.Drg, true
	case coresdk.GetDrgResponse:
		return typed.Drg, true
	case coresdk.UpdateDrgResponse:
		return typed.Drg, true
	default:
		return coresdk.Drg{}, false
	}
}

func drgLifecycleMessage(current coresdk.Drg) string {
	name := stringValue(current.DisplayName)
	if name == "" {
		name = stringValue(current.Id)
	}
	if name == "" {
		name = "Drg"
	}
	return fmt.Sprintf("Drg %s is %s", name, current.LifecycleState)
}

func buildDefaultDrgRouteTables(input corev1beta1.DrgDefaultDrgRouteTables) *coresdk.DefaultDrgRouteTables {
	if !defaultDrgRouteTablesSpecified(input) {
		return nil
	}
	output := &coresdk.DefaultDrgRouteTables{}
	if strings.TrimSpace(input.Vcn) != "" {
		output.Vcn = common.String(input.Vcn)
	}
	if strings.TrimSpace(input.IpsecTunnel) != "" {
		output.IpsecTunnel = common.String(input.IpsecTunnel)
	}
	if strings.TrimSpace(input.VirtualCircuit) != "" {
		output.VirtualCircuit = common.String(input.VirtualCircuit)
	}
	if strings.TrimSpace(input.RemotePeeringConnection) != "" {
		output.RemotePeeringConnection = common.String(input.RemotePeeringConnection)
	}
	return output
}

func statusDefaultDrgRouteTables(input *coresdk.DefaultDrgRouteTables) corev1beta1.DrgDefaultDrgRouteTables {
	if input == nil {
		return corev1beta1.DrgDefaultDrgRouteTables{}
	}
	return corev1beta1.DrgDefaultDrgRouteTables{
		Vcn:                     stringValue(input.Vcn),
		IpsecTunnel:             stringValue(input.IpsecTunnel),
		VirtualCircuit:          stringValue(input.VirtualCircuit),
		RemotePeeringConnection: stringValue(input.RemotePeeringConnection),
	}
}

func defaultDrgRouteTablesSpecified(input corev1beta1.DrgDefaultDrgRouteTables) bool {
	return strings.TrimSpace(input.Vcn) != "" ||
		strings.TrimSpace(input.IpsecTunnel) != "" ||
		strings.TrimSpace(input.VirtualCircuit) != "" ||
		strings.TrimSpace(input.RemotePeeringConnection) != ""
}

func defaultDrgRouteTablesMatch(current *coresdk.DefaultDrgRouteTables, desired corev1beta1.DrgDefaultDrgRouteTables) bool {
	if !defaultDrgRouteTablesSpecified(desired) {
		return true
	}
	if current == nil {
		return false
	}
	if strings.TrimSpace(desired.Vcn) != "" && !stringPtrEqual(current.Vcn, desired.Vcn) {
		return false
	}
	if strings.TrimSpace(desired.IpsecTunnel) != "" && !stringPtrEqual(current.IpsecTunnel, desired.IpsecTunnel) {
		return false
	}
	if strings.TrimSpace(desired.VirtualCircuit) != "" && !stringPtrEqual(current.VirtualCircuit, desired.VirtualCircuit) {
		return false
	}
	if strings.TrimSpace(desired.RemotePeeringConnection) != "" && !stringPtrEqual(current.RemotePeeringConnection, desired.RemotePeeringConnection) {
		return false
	}
	return true
}

func stringCreateOnlyMatches(actual *string, expected string) bool {
	return strings.TrimSpace(stringValue(actual)) == strings.TrimSpace(expected)
}

func stringPtrEqual(actual *string, expected string) bool {
	if actual == nil {
		return strings.TrimSpace(expected) == ""
	}
	return *actual == expected
}

func metav1Time(t time.Time) metav1.Time {
	return metav1.NewTime(t)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func sdkTimeString(value *common.SDKTime) string {
	if value == nil {
		return ""
	}
	return value.Time.Format(time.RFC3339Nano)
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func convertOCIToStatusDefinedTags(input map[string]map[string]interface{}) map[string]shared.MapValue {
	if input == nil {
		return nil
	}
	converted := make(map[string]shared.MapValue, len(input))
	for namespace, values := range input {
		convertedValues := make(shared.MapValue, len(values))
		for key, value := range values {
			convertedValues[key] = fmt.Sprint(value)
		}
		converted[namespace] = convertedValues
	}
	return converted
}
