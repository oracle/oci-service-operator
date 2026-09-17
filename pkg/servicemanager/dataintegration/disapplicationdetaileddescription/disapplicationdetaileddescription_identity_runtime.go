/* Copyright (c) 2026, Oracle and/or its affiliates. All rights reserved. */
package disapplicationdetaileddescription

import (
	"context"

	dataintegrationsdk "github.com/oracle/oci-go-sdk/v65/dataintegration"
	dataintegrationv1beta1 "github.com/oracle/oci-service-operator/api/dataintegration/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/dataintegration/detaileddescriptionruntime"
	generatedruntime "github.com/oracle/oci-service-operator/pkg/servicemanager/generatedruntime"
	shared "github.com/oracle/oci-service-operator/pkg/shared"
)

func init() {
	registerDisApplicationDetailedDescriptionRuntimeHooksMutator(func(_ *DisApplicationDetailedDescriptionServiceManager, hooks *DisApplicationDetailedDescriptionRuntimeHooks) {
		get := hooks.Get.Call
		hooks.Get.Fields = disApplicationDetailedDescriptionPathFields(hooks.Get.Fields)
		hooks.Update.Fields = disApplicationDetailedDescriptionPathFields(hooks.Update.Fields)
		hooks.Delete.Fields = disApplicationDetailedDescriptionPathFields(hooks.Delete.Fields)
		hooks.Identity = generatedruntime.IdentityHooks[*dataintegrationv1beta1.DisApplicationDetailedDescription]{
			RecordBeforeCreateFollowUp: true,
			Resolve: func(resource *dataintegrationv1beta1.DisApplicationDetailedDescription) (any, error) {
				return detaileddescriptionruntime.Resolve(resource.Spec.WorkspaceId, resource.Spec.ApplicationKey, "disApplications")
			},
			RecordTracked: func(resource *dataintegrationv1beta1.DisApplicationDetailedDescription, identity any, _ string) {
				resource.Status.OsokStatus.Ocid = shared.OCID(detaileddescriptionruntime.SyntheticID(identity.(detaileddescriptionruntime.Identity)))
			},
			LookupExisting: func(ctx context.Context, _ *dataintegrationv1beta1.DisApplicationDetailedDescription, identity any) (any, error) {
				resolved := identity.(detaileddescriptionruntime.Identity)
				return get(ctx, dataintegrationsdk.GetDisApplicationDetailedDescriptionRequest{WorkspaceId: &resolved.WorkspaceID, ApplicationKey: &resolved.ApplicationKey})
			},
			SeedSyntheticTrackedID: func(resource *dataintegrationv1beta1.DisApplicationDetailedDescription, identity any) func() {
				previous := resource.Status.OsokStatus.Ocid
				resource.Status.OsokStatus.Ocid = shared.OCID(detaileddescriptionruntime.SyntheticID(identity.(detaileddescriptionruntime.Identity)))
				return func() { resource.Status.OsokStatus.Ocid = previous }
			},
		}
	})
}

func disApplicationDetailedDescriptionPathFields(fields []generatedruntime.RequestField) []generatedruntime.RequestField {
	result := append([]generatedruntime.RequestField(nil), fields...)
	for index := range result {
		if result[index].FieldName == "ApplicationKey" {
			result[index].PreferResourceID = false
			result[index].LookupPaths = []string{"spec.applicationKey", "applicationKey"}
		}
	}
	return result
}
