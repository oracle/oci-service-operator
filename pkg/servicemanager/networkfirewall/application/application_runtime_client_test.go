/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package application

import (
	"context"
	"testing"

	"github.com/oracle/oci-go-sdk/v65/common"
	networkfirewallsdk "github.com/oracle/oci-go-sdk/v65/networkfirewall"
	networkfirewallv1beta1 "github.com/oracle/oci-service-operator/api/networkfirewall/v1beta1"
)

func TestApplicationUpdateBodyIncludesSubtypeFields(t *testing.T) {
	hooks := newApplicationDefaultRuntimeHooks(networkfirewallsdk.NetworkFirewallClient{})
	applyApplicationRuntimeHooks(&hooks)
	resource := &networkfirewallv1beta1.Application{Spec: networkfirewallv1beta1.ApplicationSpec{Type: "ICMP", IcmpType: 3, IcmpCode: 1}}
	current := networkfirewallsdk.GetApplicationResponse{Application: networkfirewallsdk.IcmpApplication{IcmpType: common.Int(8)}}
	unchanged := &networkfirewallv1beta1.Application{Spec: networkfirewallv1beta1.ApplicationSpec{Type: "ICMP", IcmpType: 8}}
	if body, update, err := hooks.BuildUpdateBody(context.Background(), unchanged, "", current); err != nil || update || body != nil {
		t.Fatalf("unchanged body=%#v update=%t err=%v", body, update, err)
	}
	body, update, err := hooks.BuildUpdateBody(context.Background(), resource, "", current)
	if err != nil {
		t.Fatal(err)
	}
	if !update {
		t.Fatal("update = false, want true")
	}
	details, ok := body.(networkfirewallv1beta1.ApplicationSpec)
	if !ok || details.IcmpType != 3 || details.IcmpCode != 1 || details.Type != "ICMP" {
		t.Fatalf("update body = %#v", body)
	}
}
