/*
  Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package decryptionprofile

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/common"
	networkfirewallsdk "github.com/oracle/oci-go-sdk/v65/networkfirewall"
	networkfirewallv1beta1 "github.com/oracle/oci-service-operator/api/networkfirewall/v1beta1"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/networkfirewall/policychildruntime"
)

func init() {
	registerDecryptionProfileRuntimeHooksMutator(func(_ *DecryptionProfileServiceManager, hooks *DecryptionProfileRuntimeHooks) {
		applyDecryptionProfileRuntimeHooks(hooks)
	})
}

func applyDecryptionProfileRuntimeHooks(hooks *DecryptionProfileRuntimeHooks) {
	if hooks == nil {
		return
	}
	getCall := hooks.Get.Call
	mutableFields := []string{"description", "type", "isUnsupportedVersionBlocked", "isUnsupportedCipherBlocked", "isOutOfCapacityBlocked", "isExpiredCertificateBlocked", "isUntrustedIssuerBlocked", "isRevocationStatusTimeoutBlocked", "isUnknownRevocationStatusBlocked", "areCertificateExtensionsRestricted", "isAutoIncludeAltName"}
	hooks.Semantics = policychildruntime.Semantics("decryptionprofile", mutableFields)
	hooks.BuildUpdateBody = func(_ context.Context, resource *networkfirewallv1beta1.DecryptionProfile, _ string, currentResponse any) (any, bool, error) {
		return policychildruntime.FullUpdateBody(resource.Spec, currentResponse, mutableFields)
	}
	hooks.Identity = generatedruntime.IdentityHooks[*networkfirewallv1beta1.DecryptionProfile]{
		RecordBeforeCreateFollowUp: true,
		Resolve: func(resource *networkfirewallv1beta1.DecryptionProfile) (any, error) {
			return policychildruntime.Resolve(resource.Spec.NetworkFirewallPolicyId, resource.Spec.Name, resource.Status.ParentResourceId, resource.Status.Name)
		},
		RecordPath: func(resource *networkfirewallv1beta1.DecryptionProfile, value any) {
			identity := value.(policychildruntime.Identity)
			resource.Status.ParentResourceId = identity.PolicyID
			resource.Status.Name = identity.Name
		},
		RecordTracked: func(resource *networkfirewallv1beta1.DecryptionProfile, value any, _ string) {
			policychildruntime.Record(&resource.Status.OsokStatus, &resource.Status.ParentResourceId, &resource.Status.Name, value.(policychildruntime.Identity))
		},
		LookupExisting: func(ctx context.Context, _ *networkfirewallv1beta1.DecryptionProfile, value any) (any, error) {
			identity := value.(policychildruntime.Identity)
			return getCall(ctx, networkfirewallsdk.GetDecryptionProfileRequest{NetworkFirewallPolicyId: common.String(identity.PolicyID), DecryptionProfileName: common.String(identity.Name)})
		},
		SeedSyntheticTrackedID: func(resource *networkfirewallv1beta1.DecryptionProfile, value any) func() {
			return policychildruntime.Seed(&resource.Status.OsokStatus, value.(policychildruntime.Identity))
		},
	}
	policyField := policychildruntime.PolicyIDField()
	nameField := policychildruntime.NameField("DecryptionProfileName", "decryptionProfileName")
	hooks.Create.Fields = []generatedruntime.RequestField{policyField, {FieldName: "CreateDecryptionProfileDetails", Contribution: "body"}}
	hooks.Get.Fields = []generatedruntime.RequestField{policyField, nameField}
	hooks.List.Fields = []generatedruntime.RequestField{policyField}
	hooks.Update.Fields = []generatedruntime.RequestField{policyField, nameField, {FieldName: "UpdateDecryptionProfileDetails", Contribution: "body"}}
	hooks.Delete.Fields = []generatedruntime.RequestField{policyField, nameField}
}
