/*
  Copyright (c) 2021, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sddc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oracle/oci-go-sdk/v65/common"
	ocvpsdk "github.com/oracle/oci-go-sdk/v65/ocvp"
	ocvpv1beta1 "github.com/oracle/oci-service-operator/api/ocvp/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
)

type sddcIdentity struct {
	compartmentID string
	displayName   string
}

type sanitizedSddcResponse struct {
	Body map[string]any `presentIn:"body"`
}

func init() {
	registerSddcRuntimeHooksMutator(func(_ *SddcServiceManager, hooks *SddcRuntimeHooks) {
		applySddcRuntimeHooks(hooks)
	})
}

func applySddcRuntimeHooks(hooks *SddcRuntimeHooks) {
	if hooks == nil {
		return
	}

	listCall := hooks.List.Call
	hooks.Identity.Resolve = func(resource *ocvpv1beta1.Sddc) (any, error) {
		return resolveSddcIdentity(resource), nil
	}
	hooks.Identity.GuardExistingBeforeCreate = guardSddcExistingBeforeCreate
	hooks.Identity.LookupExisting = func(
		ctx context.Context,
		_ *ocvpv1beta1.Sddc,
		identity any,
	) (any, error) {
		return lookupExistingSddc(ctx, listCall, identity.(sddcIdentity))
	}
	hooks.Read.Get = sddcSanitizedReadOperation(hooks.Get)
}

func guardSddcExistingBeforeCreate(
	_ context.Context,
	resource *ocvpv1beta1.Sddc,
) (generatedruntime.ExistingBeforeCreateDecision, error) {
	if sddcIdentityResolutionRequiresDisplayName(resource) {
		return generatedruntime.ExistingBeforeCreateDecisionFail, fmt.Errorf("Sddc spec.displayName is required when no OCI identifier is recorded")
	}
	return generatedruntime.ExistingBeforeCreateDecisionAllow, nil
}

func sddcSanitizedReadOperation(
	get runtimeOperationHooks[ocvpsdk.GetSddcRequest, ocvpsdk.GetSddcResponse],
) *generatedruntime.Operation {
	return &generatedruntime.Operation{
		NewRequest: func() any { return &ocvpsdk.GetSddcRequest{} },
		Fields:     append([]generatedruntime.RequestField(nil), get.Fields...),
		Call: func(ctx context.Context, request any) (any, error) {
			response, err := get.Call(ctx, *request.(*ocvpsdk.GetSddcRequest))
			if err != nil {
				return nil, err
			}
			return sanitizeSddcResponse(response.Sddc)
		},
	}
}

func lookupExistingSddc(
	ctx context.Context,
	listCall func(context.Context, ocvpsdk.ListSddcsRequest) (ocvpsdk.ListSddcsResponse, error),
	identity sddcIdentity,
) (any, error) {
	if listCall == nil || identity.compartmentID == "" || identity.displayName == "" {
		return nil, nil
	}

	response, err := listCall(ctx, ocvpsdk.ListSddcsRequest{
		CompartmentId: common.String(identity.compartmentID),
		DisplayName:   common.String(identity.displayName),
	})
	if err != nil {
		return nil, fmt.Errorf("resolve existing Sddc via ListSddcs: %w", err)
	}

	matches := make([]sanitizedSddcResponse, 0, 1)
	for _, item := range response.Items {
		if !isReusableSddcSummary(item, identity.compartmentID, identity.displayName) {
			continue
		}
		sanitized, err := sanitizeSddcSummary(item)
		if err != nil {
			return nil, err
		}
		matches = append(matches, sanitized)
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf(
			"Sddc bind lookup found multiple reusable OCI resources for compartmentId=%q displayName=%q",
			identity.compartmentID,
			identity.displayName,
		)
	}
}

func resolveSddcIdentity(resource *ocvpv1beta1.Sddc) sddcIdentity {
	if resource == nil {
		return sddcIdentity{}
	}
	return sddcIdentity{
		compartmentID: strings.TrimSpace(resource.Spec.CompartmentId),
		displayName:   strings.TrimSpace(resource.Spec.DisplayName),
	}
}

func sddcIdentityResolutionRequiresDisplayName(resource *ocvpv1beta1.Sddc) bool {
	if resource == nil {
		return false
	}
	if strings.TrimSpace(resource.Spec.DisplayName) != "" {
		return false
	}
	return currentSddcID(resource) == ""
}

func currentSddcID(resource *ocvpv1beta1.Sddc) string {
	if resource == nil {
		return ""
	}
	if ocid := strings.TrimSpace(string(resource.Status.OsokStatus.Ocid)); ocid != "" {
		return ocid
	}
	return strings.TrimSpace(resource.Status.Id)
}

func sanitizeSddcResponse(body ocvpsdk.Sddc) (sanitizedSddcResponse, error) {
	return sanitizeSddcPayload(body, "Sddc response body")
}

func sanitizeSddcSummary(body ocvpsdk.SddcSummary) (sanitizedSddcResponse, error) {
	return sanitizeSddcPayload(body, "Sddc summary body")
}

func sanitizeSddcPayload(body any, label string) (sanitizedSddcResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return sanitizedSddcResponse{}, fmt.Errorf("marshal %s: %w", label, err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return sanitizedSddcResponse{}, fmt.Errorf("decode %s: %w", label, err)
	}

	sanitized, ok := sanitizeJSONValue(decoded)
	if !ok {
		return sanitizedSddcResponse{}, nil
	}
	sanitizedMap, _ := sanitized.(map[string]any)
	return sanitizedSddcResponse{Body: sanitizedMap}, nil
}

func sanitizeJSONValue(value any) (any, bool) {
	switch concrete := value.(type) {
	case nil:
		return nil, false
	case map[string]any:
		cleaned := make(map[string]any, len(concrete))
		for key, child := range concrete {
			sanitizedChild, ok := sanitizeJSONValue(child)
			if !ok {
				continue
			}
			cleaned[key] = sanitizedChild
		}
		if len(cleaned) == 0 {
			return nil, false
		}
		return cleaned, true
	case []any:
		cleaned := make([]any, 0, len(concrete))
		for _, child := range concrete {
			sanitizedChild, ok := sanitizeJSONValue(child)
			if !ok {
				continue
			}
			cleaned = append(cleaned, sanitizedChild)
		}
		if len(cleaned) == 0 {
			return nil, false
		}
		return cleaned, true
	default:
		return concrete, true
	}
}

func isReusableSddcSummary(item ocvpsdk.SddcSummary, compartmentID string, displayName string) bool {
	if stringPointerValue(item.CompartmentId) != compartmentID {
		return false
	}
	if stringPointerValue(item.DisplayName) != displayName {
		return false
	}

	switch strings.TrimSpace(string(item.LifecycleState)) {
	case "ACTIVE", "CREATING", "UPDATING":
		return true
	default:
		return false
	}
}

func stringPointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
