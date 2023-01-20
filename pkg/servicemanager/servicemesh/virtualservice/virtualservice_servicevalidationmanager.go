/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualservice

import (
	"context"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/controller-runtime/pkg/client"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/validations"
)

//+kubebuilder:webhook:path=/validate-servicemesh-oci-oracle-com-v1beta1-virtualservice,mutating=false,failurePolicy=fail,sideEffects=None,groups=servicemesh.oci.oracle.com,resources=virtualservices,verbs=create;update;delete,versions=v1beta1,name=vs-validator.servicemesh.oci.oracle.cloud.com,admissionReviewVersions={v1,v1beta1}

type VirtualServiceValidator struct {
	resolver references.Resolver
	log      loggerutil.OSOKLogger
}

func NewVirtualServiceValidator(resolver references.Resolver, log loggerutil.OSOKLogger) manager.CustomResourceValidator {
	return &VirtualServiceValidator{resolver: resolver, log: log}
}

func (v *VirtualServiceValidator) ValidateOnCreate(context context.Context, object client.Object) (bool, string) {
	vs, err := getVirtualService(object)
	if err != nil {
		return false, err.Error()
	}

	allowed, reason := validations.IsMetadataNameValid(vs.GetName())
	if !allowed {
		return false, reason
	}

	// Only validate for the requests come from k8s operator
	if len(vs.Spec.Mesh.Id) == 0 {
		allowed, reason := validations.IsMeshPresent(v.resolver, context, vs.Spec.Mesh.ResourceRef, &vs.ObjectMeta)
		if !allowed {
			return false, reason
		}
	}

	// validate if the mtls mode supplied meets the minimum mode set on the parent mesh
	if vs.Spec.Mtls != nil {
		return v.isModeValid(context, vs)
	}

	return true, ""
}

func (v *VirtualServiceValidator) GetStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	vs, err := getVirtualService(object)
	if err != nil {
		return nil, err
	}
	return &vs.Status, nil
}

func (v *VirtualServiceValidator) ResolveRef(object client.Object) (bool, string) {
	vs, err := getVirtualService(object)
	if err != nil {
		return false, err.Error()
	}
	err = validations.ValidateMeshRef(&vs.Spec.Mesh)
	if err != nil {
		return false, err.Error()
	}
	return true, ""
}

func (v *VirtualServiceValidator) ValidateObject(object client.Object) error {
	_, err := getVirtualService(object)
	return err
}

func (v *VirtualServiceValidator) ValidateOnUpdate(context context.Context, object client.Object, oldObject client.Object) (bool, string) {
	vs, err := getVirtualService(object)
	if err != nil {
		return false, err.Error()
	}
	oldVs, err := getVirtualService(oldObject)
	if err != nil {
		return false, err.Error()
	}

	if !cmp.Equal(vs.Spec.Mesh, oldVs.Spec.Mesh) {
		return false, string(commons.MeshReferenceIsImmutable)
	}

	// Throw an error if name has changed
	if validations.IsSpecNameChanged(vs.Spec.Name, oldVs.Spec.Name) {
		return false, string(commons.NameIsImmutable)
	}

	// validate if the updated mtls mode meets the minimum mode set on the parent mesh
	if vs.Spec.Mtls != nil && !cmp.Equal(vs.Spec.Mtls.Mode, oldVs.Status.VirtualServiceMtls.Mode) {
		allowed, reason := v.isModeValid(context, vs)
		if !allowed {
			return false, reason
		}
	}

	// validate if there's no virtual deployment has listeners when customer empty the hosts
	oldHasHosts := oldVs.Spec.Hosts != nil && len(oldVs.Spec.Hosts) > 0
	hasHosts := vs.Spec.Hosts != nil && len(vs.Spec.Hosts) > 0
	if oldHasHosts && !hasHosts && v.HasVDWithListeners(context, oldVs) {
		return false, string(commons.VirtualServiceHostsShouldNotBeEmpty)
	}

	return true, ""
}

func (v *VirtualServiceValidator) isModeValid(ctx context.Context, virtualService *servicemeshapi.VirtualService) (bool, string) {
	referredMeshMode, err := v.getMeshMode(ctx, virtualService)
	if len(referredMeshMode) == 0 {
		return false, err
	}

	if commons.MtlsLevel[virtualService.Spec.Mtls.Mode] < commons.MtlsLevel[referredMeshMode] {
		return false, string(commons.VirtualServiceMtlsNotSatisfied)
	}
	return true, ""
}

func (v *VirtualServiceValidator) getMeshMode(ctx context.Context, virtualService *servicemeshapi.VirtualService) (servicemeshapi.MutualTransportLayerSecurityModeEnum, string) {
	var referredMeshMode servicemeshapi.MutualTransportLayerSecurityModeEnum
	if len(virtualService.Spec.Mesh.Id) == 0 {
		resourceRef := v.resolver.ResolveResourceRef(virtualService.Spec.Mesh.ResourceRef, &virtualService.ObjectMeta)
		referredMesh, err := v.resolver.ResolveMeshReference(ctx, resourceRef)
		if err != nil {
			// referred Mesh was not found, continuing to create child resource
			return servicemeshapi.MutualTransportLayerSecurityModeDisabled, ""
		}
		if cmp.Equal(referredMesh.Status, servicemeshapi.ServiceMeshStatus{}) || referredMesh.Status.MeshMtls == nil {
			if referredMesh.Spec.Mtls != nil {
				referredMeshMode = referredMesh.Spec.Mtls.Minimum
			} else {
				referredMeshMode = servicemeshapi.MutualTransportLayerSecurityModeStrict
			}
		} else {
			referredMeshMode = referredMesh.Status.MeshMtls.Minimum
		}
	} else {
		referredMesh, err := v.resolver.ResolveMeshRefById(ctx, &virtualService.Spec.Mesh.Id)
		if err != nil {
			return "", string(commons.MeshReferenceOCIDNotFound)
		}
		referredMeshMode = referredMesh.Mtls.Minimum
	}
	return referredMeshMode, ""
}

func (v *VirtualServiceValidator) GetEntityType() client.Object {
	return &servicemeshapi.VirtualService{}
}

func (v *VirtualServiceValidator) HasVDWithListeners(ctx context.Context, vs *servicemeshapi.VirtualService) bool {
	hasVdWithListeners, err := v.resolver.ResolveHasVirtualDeploymentWithListener(ctx, &vs.Spec.CompartmentId, &vs.Status.VirtualServiceId)
	if err != nil {
		v.log.ErrorLog(err, "Failed to resolve the virtual deployments under virtual service")
	}

	return hasVdWithListeners
}
