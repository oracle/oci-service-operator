/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

// Package policychildruntime contains the common path-identity contract for
// named resources owned by one Network Firewall policy.
package policychildruntime

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

// Identity addresses a named child beneath a Network Firewall policy.
type Identity struct {
	PolicyID string
	Name     string
}

// Resolve prefers projected status identity and falls back to desired spec.
func Resolve(specPolicyID, specName, statusPolicyID, statusName string) (Identity, error) {
	identity := Identity{
		PolicyID: firstNonEmpty(statusPolicyID, specPolicyID),
		Name:     firstNonEmpty(statusName, specName),
	}
	if identity.PolicyID == "" {
		return Identity{}, fmt.Errorf("resolve Network Firewall policy child identity: networkFirewallPolicyId is empty")
	}
	if identity.Name == "" {
		return Identity{}, fmt.Errorf("resolve Network Firewall policy child identity: name is empty")
	}
	return identity, nil
}

// Record persists path identity and a stable synthetic tracked ID.
func Record(status *shared.OSOKStatus, parentResourceID, name *string, identity Identity) {
	if parentResourceID != nil {
		*parentResourceID = identity.PolicyID
	}
	if name != nil {
		*name = identity.Name
	}
	if status != nil {
		status.Ocid = shared.OCID(SyntheticID(identity))
	}
}

// Seed temporarily supplies the stable tracked ID while a path-addressed OCI call runs.
func Seed(status *shared.OSOKStatus, identity Identity) func() {
	if status == nil {
		return func() {}
	}
	previous := status.Ocid
	status.Ocid = shared.OCID(SyntheticID(identity))
	return func() { status.Ocid = previous }
}

// SyntheticID scopes a child name to its owning policy.
func SyntheticID(identity Identity) string {
	return strings.TrimSpace(identity.PolicyID) + "/" + strings.TrimSpace(identity.Name)
}

// PolicyIDField maps the parent path parameter from status or spec.
func PolicyIDField() generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    "NetworkFirewallPolicyId",
		RequestName:  "networkFirewallPolicyId",
		Contribution: "path",
		LookupPaths:  []string{"status.parentResourceId", "spec.networkFirewallPolicyId"},
	}
}

// NameField maps a resource-specific path parameter from the common child name.
func NameField(fieldName, requestName string) generatedruntime.RequestField {
	return generatedruntime.RequestField{
		FieldName:    fieldName,
		RequestName:  requestName,
		Contribution: "path",
		LookupPaths:  []string{"status.name", "spec.name", "name"},
	}
}

// Semantics declares immediate CRUD with readback and confirmed deletion.
func Semantics(slug string, mutable []string) *generatedruntime.Semantics {
	return &generatedruntime.Semantics{
		FormalService:     "networkfirewall",
		FormalSlug:        slug,
		StatusProjection:  "required",
		SecretSideEffects: "none",
		FinalizerPolicy:   "retain-until-confirmed-delete",
		Async: &generatedruntime.AsyncSemantics{
			Strategy:             "none",
			Runtime:              "generatedruntime",
			FormalClassification: "none",
		},
		Lifecycle: generatedruntime.LifecycleSemantics{ActiveStates: []string{"ACTIVE"}},
		Delete: generatedruntime.DeleteSemantics{
			Policy:         "required",
			TerminalStates: []string{"DELETED"},
		},
		List: &generatedruntime.ListSemantics{
			ResponseItemsField: "Items",
			MatchFields:        []string{"parentResourceId", "name"},
		},
		Mutation: generatedruntime.MutationSemantics{
			Mutable:       append([]string(nil), mutable...),
			ForceNew:      []string{"networkFirewallPolicyId", "name"},
			ConflictsWith: map[string][]string{},
		},
		CreateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		UpdateFollowUp: generatedruntime.FollowUpSemantics{Strategy: "read-after-write"},
		DeleteFollowUp: generatedruntime.FollowUpSemantics{Strategy: "confirm-delete"},
	}
}

// FullUpdateBody compares the declared mutable fields and returns the complete
// desired spec when OCI's update model requires mandatory subtype fields.
func FullUpdateBody(spec any, currentResponse any, mutable []string) (any, bool, error) {
	current, ok := responseBody(currentResponse)
	if !ok {
		return spec, true, nil
	}
	for _, path := range mutable {
		desired, desiredFound := valueByPath(reflect.ValueOf(spec), path)
		if !desiredFound {
			continue
		}
		observed, observedFound := valueByPath(current, path)
		if !observedFound {
			observed, observedFound = jsonValueByPath(current, path)
		}
		if !observedFound {
			return spec, true, nil
		}
		if !observed.IsValid() && desired.IsValid() && desired.IsZero() {
			continue
		}
		equal, err := normalizedEqual(desired, observed)
		if err != nil {
			return nil, false, fmt.Errorf("compare Network Firewall policy child field %s: %w", path, err)
		}
		if !equal {
			return spec, true, nil
		}
	}
	return nil, false, nil
}

func jsonValueByPath(value reflect.Value, path string) (reflect.Value, bool) {
	if !value.IsValid() {
		return reflect.Value{}, false
	}
	payload, err := json.Marshal(value.Interface())
	if err != nil {
		return reflect.Value{}, false
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return reflect.Value{}, false
	}
	current := decoded
	for _, segment := range strings.Split(path, ".") {
		values, ok := current.(map[string]any)
		if !ok {
			return reflect.Value{}, false
		}
		key := ""
		for candidate := range values {
			if normalize(candidate) == normalize(segment) {
				key = candidate
				break
			}
		}
		if key == "" {
			return reflect.Value{}, false
		}
		current = values[key]
	}
	if current == nil {
		return reflect.Value{}, true
	}
	return reflect.ValueOf(current), true
}

func responseBody(response any) (reflect.Value, bool) {
	value := dereference(reflect.ValueOf(response))
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	typeOf := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typeOf.Field(i)
		if !field.Anonymous || field.Name == "RawResponse" {
			continue
		}
		candidate := dereference(value.Field(i))
		if candidate.IsValid() && candidate.Kind() == reflect.Struct {
			return candidate, true
		}
	}
	return value, true
}

func valueByPath(value reflect.Value, path string) (reflect.Value, bool) {
	current := dereference(value)
	for _, segment := range strings.Split(path, ".") {
		segment = strings.TrimSpace(segment)
		if segment == "" || !current.IsValid() || current.Kind() != reflect.Struct {
			return reflect.Value{}, false
		}
		field, ok := structField(current, segment)
		if !ok {
			return reflect.Value{}, false
		}
		current = dereference(field)
	}
	if !current.IsValid() {
		return reflect.Value{}, true
	}
	return current, true
}

func structField(value reflect.Value, segment string) (reflect.Value, bool) {
	normalized := normalize(segment)
	typeOf := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := typeOf.Field(i)
		if !field.IsExported() {
			continue
		}
		jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
		if normalize(field.Name) == normalized || normalize(jsonName) == normalized {
			return value.Field(i), true
		}
	}
	return reflect.Value{}, false
}

func dereference(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func normalizedEqual(left, right reflect.Value) (bool, error) {
	leftValue, err := normalizedJSONValue(left)
	if err != nil {
		return false, err
	}
	rightValue, err := normalizedJSONValue(right)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(leftValue, rightValue), nil
}

func normalizedJSONValue(value reflect.Value) (any, error) {
	if !value.IsValid() {
		return nil, nil
	}
	payload, err := json.Marshal(value.Interface())
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return pruneNullChildren(decoded), nil
}

func pruneNullChildren(value any) any {
	switch concrete := value.(type) {
	case map[string]any:
		pruned := make(map[string]any, len(concrete))
		for key, child := range concrete {
			if child == nil {
				continue
			}
			pruned[key] = pruneNullChildren(child)
		}
		return pruned
	case []any:
		pruned := make([]any, len(concrete))
		for i, child := range concrete {
			pruned[i] = pruneNullChildren(child)
		}
		return pruned
	default:
		return concrete
	}
}

func normalize(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	return strings.ToLower(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
