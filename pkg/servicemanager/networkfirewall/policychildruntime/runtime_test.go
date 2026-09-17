/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package policychildruntime

import (
	"reflect"
	"testing"

	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func TestResolveRecordAndSeed(t *testing.T) {
	identity, err := Resolve("spec-policy", "spec-name", "status-policy", "status-name")
	if err != nil {
		t.Fatal(err)
	}
	if identity != (Identity{PolicyID: "status-policy", Name: "status-name"}) {
		t.Fatalf("identity = %+v", identity)
	}
	status := shared.OSOKStatus{}
	var parent, name string
	Record(&status, &parent, &name, identity)
	if parent != identity.PolicyID || name != identity.Name || string(status.Ocid) != "status-policy/status-name" {
		t.Fatalf("recorded identity = parent:%q name:%q status:%+v", parent, name, status)
	}
	restore := Seed(&status, Identity{PolicyID: "temporary-policy", Name: "temporary-name"})
	if string(status.Ocid) != "temporary-policy/temporary-name" {
		t.Fatalf("seeded OCID = %q", status.Ocid)
	}
	restore()
	if string(status.Ocid) != "status-policy/status-name" {
		t.Fatalf("restored OCID = %q", status.Ocid)
	}
}

func TestFieldsAndSemantics(t *testing.T) {
	policyField := PolicyIDField()
	nameField := NameField("AddressListName", "addressListName")
	if !reflect.DeepEqual(policyField.LookupPaths, []string{"status.parentResourceId", "spec.networkFirewallPolicyId"}) {
		t.Fatalf("policy lookup paths = %v", policyField.LookupPaths)
	}
	if !reflect.DeepEqual(nameField.LookupPaths, []string{"status.name", "spec.name", "name"}) {
		t.Fatalf("name lookup paths = %v", nameField.LookupPaths)
	}
	semantics := Semantics("addresslist", []string{"addresses", "description"})
	if semantics.Async.Strategy != "none" || semantics.DeleteFollowUp.Strategy != "confirm-delete" {
		t.Fatalf("semantics = %+v", semantics)
	}
}

func TestFullUpdateBodyIncludesRequiredFieldsOnlyOnDrift(t *testing.T) {
	type desired struct {
		Type        string `json:"type,omitempty"`
		IcmpType    int    `json:"icmpType,omitempty"`
		IcmpCode    int    `json:"icmpCode,omitempty"`
		Description string `json:"description,omitempty"`
		Blocked     bool   `json:"blocked,omitempty"`
	}
	type observed struct {
		Type        string
		IcmpType    *int
		IcmpCode    *int
		Description *string
		Blocked     *bool
	}
	type response struct{ observed }
	intValue, description, falseValue := 8, "original", false
	spec := desired{Type: "ICMP", IcmpType: 8, Description: "original"}
	current := response{observed: observed{Type: "ICMP", IcmpType: &intValue, Description: &description, Blocked: &falseValue}}
	body, update, err := FullUpdateBody(spec, current, []string{"type", "icmpType", "icmpCode", "description", "blocked"})
	if err != nil || update || body != nil {
		t.Fatalf("unchanged body=%#v update=%t err=%v", body, update, err)
	}
	spec.Description = "updated"
	body, update, err = FullUpdateBody(spec, current, []string{"type", "icmpType", "icmpCode", "description", "blocked"})
	if err != nil || !update || !reflect.DeepEqual(body, spec) {
		t.Fatalf("updated body=%#v update=%t err=%v", body, update, err)
	}
}

func TestFullUpdateBodyTreatsEmptyObjectAndNullChildrenAsEqual(t *testing.T) {
	type position struct {
		Before string `json:"before,omitempty"`
		After  string `json:"after,omitempty"`
	}
	type desired struct{ Position position }
	type observedPosition struct {
		Before *string `json:"before"`
		After  *string `json:"after"`
	}
	type observed struct{ Position observedPosition }
	type response struct{ observed }
	body, update, err := FullUpdateBody(desired{}, response{}, []string{"position"})
	if err != nil || update || body != nil {
		t.Fatalf("body=%#v update=%t err=%v", body, update, err)
	}
}
