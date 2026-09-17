/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package generatedruntime

import (
	"reflect"
	"testing"

	networkfirewallsdk "github.com/oracle/oci-go-sdk/v65/networkfirewall"
)

func TestConvertValueSupportsNetworkFirewallPolymorphicBodies(t *testing.T) {
	tests := []struct {
		name       string
		payload    map[string]any
		targetType reflect.Type
		wantType   reflect.Type
	}{
		{name: "update IP address list", payload: map[string]any{"type": "IP", "addresses": []string{"10.0.0.0/24"}}, targetType: networkFirewallUpdateAddressListType, wantType: reflect.TypeOf(networkfirewallsdk.UpdateIpAddressListDetails{})},
		{name: "create ICMP application", payload: map[string]any{"type": "ICMP", "name": "icmp", "icmpType": 8}, targetType: networkFirewallCreateApplicationType, wantType: reflect.TypeOf(networkfirewallsdk.CreateIcmpApplicationDetails{})},
		{name: "update ICMPv6 application", payload: map[string]any{"type": "ICMP_V6", "icmpType": 128}, targetType: networkFirewallUpdateApplicationType, wantType: reflect.TypeOf(networkfirewallsdk.UpdateIcmp6ApplicationDetails{})},
		{name: "create forward proxy profile", payload: map[string]any{"type": "SSL_FORWARD_PROXY", "name": "forward"}, targetType: networkFirewallCreateDecryptionType, wantType: reflect.TypeOf(networkfirewallsdk.CreateSslForwardProxyProfileDetails{})},
		{name: "update inbound profile", payload: map[string]any{"type": "SSL_INBOUND_INSPECTION"}, targetType: networkFirewallUpdateDecryptionType, wantType: reflect.TypeOf(networkfirewallsdk.UpdateSslInboundInspectionProfileDetails{})},
		{name: "create mapped secret", payload: map[string]any{"source": "OCI_VAULT", "name": "secret", "vaultSecretId": "ocid1.vaultsecret.oc1..test", "versionNumber": 1}, targetType: networkFirewallCreateMappedSecretType, wantType: reflect.TypeOf(networkfirewallsdk.CreateVaultMappedSecretDetails{})},
		{name: "update mapped secret", payload: map[string]any{"source": "OCI_VAULT", "vaultSecretId": "ocid1.vaultsecret.oc1..test", "versionNumber": 2}, targetType: networkFirewallUpdateMappedSecretType, wantType: reflect.TypeOf(networkfirewallsdk.UpdateVaultMappedSecretDetails{})},
		{name: "create NAT rule", payload: map[string]any{"type": "NATV4", "name": "nat", "action": "DYNAMIC_IP_AND_PORT", "position": map[string]any{}}, targetType: networkFirewallCreateNatRuleType, wantType: reflect.TypeOf(networkfirewallsdk.CreateNatV4RuleDetails{})},
		{name: "update NAT rule", payload: map[string]any{"type": "NATV4", "action": "DYNAMIC_IP_AND_PORT", "position": map[string]any{}}, targetType: networkFirewallUpdateNatRuleType, wantType: reflect.TypeOf(networkfirewallsdk.UpdateNatV4RuleDetails{})},
		{name: "create TCP service", payload: map[string]any{"type": "TCP_SERVICE", "name": "tcp", "portRanges": []map[string]any{{"minimumPort": 80, "maximumPort": 80}}}, targetType: networkFirewallCreateServiceType, wantType: reflect.TypeOf(networkfirewallsdk.CreateTcpServiceDetails{})},
		{name: "update UDP service", payload: map[string]any{"type": "UDP_SERVICE", "portRanges": []map[string]any{{"minimumPort": 53, "maximumPort": 53}}}, targetType: networkFirewallUpdateServiceType, wantType: reflect.TypeOf(networkfirewallsdk.UpdateUdpServiceDetails{})},
		{name: "create VXLAN rule", payload: map[string]any{"protocol": "VXLAN", "name": "vxlan", "action": "INSPECT", "position": map[string]any{}}, targetType: networkFirewallCreateTunnelRuleType, wantType: reflect.TypeOf(networkfirewallsdk.CreateVxlanInspectionRuleDetails{})},
		{name: "update VXLAN rule", payload: map[string]any{"protocol": "VXLAN", "action": "INSPECT", "position": map[string]any{}}, targetType: networkFirewallUpdateTunnelRuleType, wantType: reflect.TypeOf(networkfirewallsdk.UpdateVxlanInspectionRuleDetails{})},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			converted, err := convertValue(test.payload, test.targetType)
			if err != nil {
				t.Fatal(err)
			}
			if got := reflect.TypeOf(converted.Interface()); got != test.wantType {
				t.Fatalf("converted type = %v, want %v", got, test.wantType)
			}
		})
	}
}

func TestConvertValueRejectsUnsupportedNetworkFirewallDiscriminator(t *testing.T) {
	_, err := convertValue(map[string]any{"type": "UNKNOWN"}, networkFirewallCreateServiceType)
	if err == nil {
		t.Fatal("expected unsupported Network Firewall discriminator error")
	}
}

func TestPreserveNetworkFirewallUpdateDiscriminator(t *testing.T) {
	tests := []struct {
		name          string
		request       func() any
		field         RequestField
		discriminator string
		value         string
	}{
		{name: "type", request: func() any { return &networkfirewallsdk.UpdateServiceRequest{} }, field: RequestField{FieldName: "UpdateServiceDetails", Contribution: "body"}, discriminator: "type", value: "TCP_SERVICE"},
		{name: "source", request: func() any { return &networkfirewallsdk.UpdateMappedSecretRequest{} }, field: RequestField{FieldName: "UpdateMappedSecretDetails", Contribution: "body"}, discriminator: "source", value: "OCI_VAULT"},
		{name: "protocol", request: func() any { return &networkfirewallsdk.UpdateTunnelInspectionRuleRequest{} }, field: RequestField{FieldName: "UpdateTunnelInspectionRuleDetails", Contribution: "body"}, discriminator: "protocol", value: "VXLAN"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := map[string]any{"description": "updated"}
			preservePolymorphicUpdateDiscriminator(body, map[string]any{test.discriminator: test.value}, &Operation{NewRequest: test.request, Fields: []RequestField{test.field}})
			if body[test.discriminator] != test.value {
				t.Fatalf("body = %#v", body)
			}
		})
	}
}
