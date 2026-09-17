/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package sensitivetype

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	datasafesdk "github.com/oracle/oci-go-sdk/v65/datasafe"
	datasafev1beta1 "github.com/oracle/oci-service-operator/api/datasafe/v1beta1"
)

var sensitiveTypeMutableFields = []string{
	"displayName",
	"shortName",
	"description",
	"parentCategoryId",
	"freeformTags",
	"definedTags",
	"namePattern",
	"commentPattern",
	"dataPattern",
	"defaultMaskingFormatId",
	"searchType",
}

func init() {
	registerSensitiveTypeRuntimeHooksMutator(func(_ *SensitiveTypeServiceManager, hooks *SensitiveTypeRuntimeHooks) {
		hooks.BuildCreateBody = buildSensitiveTypeCreateBody
		hooks.BuildUpdateBody = buildSensitiveTypeUpdateBody
	})
}

func buildSensitiveTypeCreateBody(
	_ context.Context,
	resource *datasafev1beta1.SensitiveType,
	_ string,
) (any, error) {
	if resource == nil {
		return nil, fmt.Errorf("SensitiveType resource is nil")
	}
	payload, err := json.Marshal(resource.Spec)
	if err != nil {
		return nil, fmt.Errorf("marshal SensitiveType spec: %w", err)
	}

	switch strings.ToUpper(strings.TrimSpace(resource.Spec.EntityType)) {
	case "SENSITIVE_TYPE":
		var details datasafesdk.CreateSensitiveTypePatternDetails
		if err := json.Unmarshal(payload, &details); err != nil {
			return nil, fmt.Errorf("build sensitive-type pattern create details: %w", err)
		}
		return details, nil
	case "SENSITIVE_CATEGORY":
		var details datasafesdk.CreateSensitiveCategoryDetails
		if err := json.Unmarshal(payload, &details); err != nil {
			return nil, fmt.Errorf("build sensitive-category create details: %w", err)
		}
		return details, nil
	default:
		return nil, fmt.Errorf("SensitiveType spec.entityType %q must be SENSITIVE_TYPE or SENSITIVE_CATEGORY", resource.Spec.EntityType)
	}
}

func buildSensitiveTypeUpdateBody(
	_ context.Context,
	resource *datasafev1beta1.SensitiveType,
	_ string,
	currentResponse any,
) (any, bool, error) {
	if resource == nil {
		return nil, false, fmt.Errorf("SensitiveType resource is nil")
	}
	payload, err := json.Marshal(resource.Spec)
	if err != nil {
		return nil, false, fmt.Errorf("marshal SensitiveType spec: %w", err)
	}
	updateNeeded, err := sensitiveTypeUpdateNeeded(payload, currentResponse)
	if err != nil {
		return nil, false, err
	}

	switch strings.ToUpper(strings.TrimSpace(resource.Spec.EntityType)) {
	case "SENSITIVE_TYPE":
		var details datasafesdk.UpdateSensitiveTypePatternDetails
		if err := json.Unmarshal(payload, &details); err != nil {
			return nil, false, fmt.Errorf("build sensitive-type pattern update details: %w", err)
		}
		return details, updateNeeded, nil
	case "SENSITIVE_CATEGORY":
		var details datasafesdk.UpdateSensitiveCategoryDetails
		if err := json.Unmarshal(payload, &details); err != nil {
			return nil, false, fmt.Errorf("build sensitive-category update details: %w", err)
		}
		return details, updateNeeded, nil
	default:
		return nil, false, fmt.Errorf("SensitiveType spec.entityType %q must be SENSITIVE_TYPE or SENSITIVE_CATEGORY", resource.Spec.EntityType)
	}
}

func sensitiveTypeUpdateNeeded(desiredPayload []byte, currentResponse any) (bool, error) {
	current, ok := sensitiveTypeResponseBody(currentResponse)
	if !ok {
		return false, fmt.Errorf("current SensitiveType response does not expose a SensitiveType body")
	}
	currentPayload, err := json.Marshal(current)
	if err != nil {
		return false, fmt.Errorf("marshal current SensitiveType: %w", err)
	}
	desiredFields := map[string]any{}
	currentFields := map[string]any{}
	if err := json.Unmarshal(desiredPayload, &desiredFields); err != nil {
		return false, fmt.Errorf("decode desired SensitiveType fields: %w", err)
	}
	if err := json.Unmarshal(currentPayload, &currentFields); err != nil {
		return false, fmt.Errorf("decode current SensitiveType fields: %w", err)
	}
	for _, field := range sensitiveTypeMutableFields {
		desired, specified := desiredFields[field]
		if specified && !reflect.DeepEqual(desired, currentFields[field]) {
			return true, nil
		}
	}
	return false, nil
}

func sensitiveTypeResponseBody(response any) (any, bool) {
	switch typed := response.(type) {
	case datasafesdk.GetSensitiveTypeResponse:
		return typed.SensitiveType, typed.SensitiveType != nil
	case *datasafesdk.GetSensitiveTypeResponse:
		if typed == nil {
			return nil, false
		}
		return typed.SensitiveType, typed.SensitiveType != nil
	case datasafesdk.CreateSensitiveTypeResponse:
		return typed.SensitiveType, typed.SensitiveType != nil
	case *datasafesdk.CreateSensitiveTypeResponse:
		if typed == nil {
			return nil, false
		}
		return typed.SensitiveType, typed.SensitiveType != nil
	case datasafesdk.SensitiveType:
		return typed, typed != nil
	default:
		return nil, false
	}
}
