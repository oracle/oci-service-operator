/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package addresslist

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/common"
	networkfirewallsdk "github.com/oracle/oci-go-sdk/v65/networkfirewall"
	networkfirewallv1beta1 "github.com/oracle/oci-service-operator/api/networkfirewall/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/networkfirewall/policychildruntime"
)

func init() {
	registerAddressListRuntimeHooksMutator(func(_ *AddressListServiceManager, hooks *AddressListRuntimeHooks) {
		applyAddressListRuntimeHooks(hooks)
	})
}

func applyAddressListRuntimeHooks(hooks *AddressListRuntimeHooks) {
	if hooks == nil {
		return
	}
	getCall := hooks.Get.Call
	mutableFields := []string{"type", "addresses", "description"}
	hooks.Semantics = policychildruntime.Semantics("addresslist", mutableFields)
	hooks.BuildUpdateBody = func(_ context.Context, resource *networkfirewallv1beta1.AddressList, _ string, currentResponse any) (any, bool, error) {
		return policychildruntime.FullUpdateBody(resource.Spec, currentResponse, mutableFields)
	}
	hooks.Identity = generatedruntime.IdentityHooks[*networkfirewallv1beta1.AddressList]{
		RecordBeforeCreateFollowUp: true,
		Resolve: func(resource *networkfirewallv1beta1.AddressList) (any, error) {
			return policychildruntime.Resolve(resource.Spec.NetworkFirewallPolicyId, resource.Spec.Name, resource.Status.ParentResourceId, resource.Status.Name)
		},
		RecordPath: func(resource *networkfirewallv1beta1.AddressList, value any) {
			identity := value.(policychildruntime.Identity)
			resource.Status.ParentResourceId = identity.PolicyID
			resource.Status.Name = identity.Name
		},
		RecordTracked: func(resource *networkfirewallv1beta1.AddressList, value any, _ string) {
			policychildruntime.Record(&resource.Status.OsokStatus, &resource.Status.ParentResourceId, &resource.Status.Name, value.(policychildruntime.Identity))
		},
		LookupExisting: func(ctx context.Context, _ *networkfirewallv1beta1.AddressList, value any) (any, error) {
			identity := value.(policychildruntime.Identity)
			return getCall(ctx, networkfirewallsdk.GetAddressListRequest{NetworkFirewallPolicyId: common.String(identity.PolicyID), AddressListName: common.String(identity.Name)})
		},
		SeedSyntheticTrackedID: func(resource *networkfirewallv1beta1.AddressList, value any) func() {
			return policychildruntime.Seed(&resource.Status.OsokStatus, value.(policychildruntime.Identity))
		},
	}
	policyField := policychildruntime.PolicyIDField()
	nameField := policychildruntime.NameField("AddressListName", "addressListName")
	hooks.Create.Fields = []generatedruntime.RequestField{policyField, {FieldName: "CreateAddressListDetails", Contribution: "body"}}
	hooks.Get.Fields = []generatedruntime.RequestField{policyField, nameField}
	hooks.List.Fields = []generatedruntime.RequestField{policyField}
	hooks.Update.Fields = []generatedruntime.RequestField{policyField, nameField, {FieldName: "UpdateAddressListDetails", Contribution: "body"}}
	hooks.Delete.Fields = []generatedruntime.RequestField{policyField, nameField}
}
