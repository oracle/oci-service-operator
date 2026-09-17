/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package savedquery

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	cloudguardsdk "github.com/oracle/oci-go-sdk/v65/cloudguard"
	cloudguardv1beta1 "github.com/oracle/oci-service-operator/api/cloudguard/v1beta1"
)

func init() {
	registerSavedQueryRuntimeHooksMutator(func(_ *SavedQueryServiceManager, hooks *SavedQueryRuntimeHooks) {
		hooks.BuildUpdateBody = buildSavedQueryUpdateBody
	})
}

// buildSavedQueryUpdateBody preserves the full provider-proven update request
// while using the live response to avoid no-op PUTs.
func buildSavedQueryUpdateBody(
	_ context.Context,
	resource *cloudguardv1beta1.SavedQuery,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("SavedQuery resource is nil")
	}
	current, ok := savedQueryResponseBody(currentResponse)
	if !ok {
		return nil, false, fmt.Errorf("current SavedQuery response does not expose a SavedQuery body")
	}
	payload, err := json.Marshal(resource.Spec)
	if err != nil {
		return nil, false, fmt.Errorf("marshal SavedQuery spec: %w", err)
	}
	details := cloudguardsdk.UpdateSavedQueryDetails{}
	if err := json.Unmarshal(payload, &details); err != nil {
		return nil, false, fmt.Errorf("build SavedQuery update details: %w", err)
	}
	return details, savedQueryUpdateNeeded(resource.Spec, current), nil
}

func savedQueryUpdateNeeded(spec cloudguardv1beta1.SavedQuerySpec, current cloudguardsdk.SavedQuery) bool {
	if current.DisplayName == nil || *current.DisplayName != spec.DisplayName ||
		current.Query == nil || *current.Query != spec.Query {
		return true
	}
	if spec.Description != "" && (current.Description == nil || *current.Description != spec.Description) {
		return true
	}
	if spec.FreeformTags != nil && !reflect.DeepEqual(spec.FreeformTags, current.FreeformTags) {
		return true
	}
	if spec.DefinedTags != nil {
		payload, err := json.Marshal(spec.DefinedTags)
		if err != nil {
			return true
		}
		definedTags := map[string]map[string]interface{}{}
		if err := json.Unmarshal(payload, &definedTags); err != nil || !reflect.DeepEqual(definedTags, current.DefinedTags) {
			return true
		}
	}
	return false
}

func savedQueryResponseBody(response any) (cloudguardsdk.SavedQuery, bool) {
	switch typed := response.(type) {
	case cloudguardsdk.GetSavedQueryResponse:
		return typed.SavedQuery, true
	case *cloudguardsdk.GetSavedQueryResponse:
		if typed != nil {
			return typed.SavedQuery, true
		}
	case cloudguardsdk.CreateSavedQueryResponse:
		return typed.SavedQuery, true
	case *cloudguardsdk.CreateSavedQueryResponse:
		if typed != nil {
			return typed.SavedQuery, true
		}
	case cloudguardsdk.SavedQuery:
		return typed, true
	case *cloudguardsdk.SavedQuery:
		if typed != nil {
			return *typed, true
		}
	}
	return cloudguardsdk.SavedQuery{}, false
}
