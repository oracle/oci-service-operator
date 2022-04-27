/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package mesh

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

//+kubebuilder:webhook:path=/validate-servicemesh-oci-oracle-com-v1beta1-mesh,mutating=false,failurePolicy=fail,sideEffects=None,groups=servicemesh.oci.oracle.com,resources=meshes,verbs=create;update;delete,versions=v1beta1,name=mesh-validator.servicemesh.oci.oracle.cloud.com,admissionReviewVersions={v1,v1beta1}

type MeshValidator struct {
	resolver references.Resolver
	log      loggerutil.OSOKLogger
}

func NewMeshValidator(resolver references.Resolver, log loggerutil.OSOKLogger) manager.CustomResourceValidator {
	return &MeshValidator{resolver: resolver, log: log}
}

func (v *MeshValidator) ValidateOnCreate(context context.Context, object client.Object) (bool, string) {
	mesh, err := getMesh(object)
	if err != nil {
		return false, err.Error()
	}

	allowed, reason := validations.IsMetadataNameValid(mesh.GetName())
	if !allowed {
		return false, reason
	}

	return true, ""
}

func (v *MeshValidator) GetStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	mesh, err := getMesh(object)
	if err != nil {
		return nil, err
	}
	return &mesh.Status, nil
}

func (v *MeshValidator) ResolveRef(object client.Object) (bool, string) {
	return true, ""
}

func (v *MeshValidator) ValidateObject(object client.Object) error {
	_, err := getMesh(object)
	return err
}

func (v *MeshValidator) ValidateOnUpdate(context context.Context, object client.Object, oldObject client.Object) (bool, string) {
	mesh, err := getMesh(object)
	if err != nil {
		return false, err.Error()
	}
	oldMesh, err := getMesh(oldObject)
	if err != nil {
		return false, err.Error()
	}

	// Throw an error if certificate authorities has changed
	// For GA we will only support one CA ID, but we decided to make it a list in case we later add support
	// for migrating from an old CA to a new one. This validation has to be updated accordingly.
	if !cmp.Equal(mesh.Spec.CertificateAuthorities, oldMesh.Spec.CertificateAuthorities) {
		return false, string(commons.CertificateAuthoritiesIsImmutable)
	}

	// Check if the virtual services meet minimum mode before updating mesh
	if mesh.Spec.Mtls != nil && !cmp.Equal(mesh.Spec.Mtls.Minimum, oldMesh.Status.MeshMtls.Minimum) {
		allowed, reason := v.isModeValid(context, mesh, oldMesh)
		if !allowed {
			return false, reason
		}
	}

	return true, ""
}

func (v *MeshValidator) isModeValid(ctx context.Context, mesh *servicemeshapi.Mesh, oldMesh *servicemeshapi.Mesh) (bool, string) {
	virtualServiceList, err := v.resolver.ResolveVirtualServiceListByNamespace(ctx, mesh.Namespace)
	if err != nil {
		return false, "error resolving virtual services for the mesh"
	}

	// match the mesh then check if the mode meets requirement
	for _, vs := range virtualServiceList.Items {
		referredMeshId, err := v.resolver.ResolveMeshId(ctx, &vs.Spec.Mesh, &vs.ObjectMeta)
		if err != nil {
			return false, "error resolving mesh"
		}

		if *referredMeshId != oldMesh.Status.MeshId || !vs.DeletionTimestamp.IsZero() {
			continue
		}

		var vsMode servicemeshapi.MutualTransportLayerSecurityModeEnum
		if cmp.Equal(vs.Status, servicemeshapi.ServiceMeshStatus{}) || vs.Status.VirtualServiceMtls == nil {
			if vs.Spec.Mtls != nil {
				vsMode = vs.Spec.Mtls.Mode
			} else {
				vsMode = oldMesh.Status.MeshMtls.Minimum
			}
		} else {
			vsMode = vs.Status.VirtualServiceMtls.Mode
		}
		if commons.MtlsLevel[vsMode] < commons.MtlsLevel[mesh.Spec.Mtls.Minimum] {
			return false, string(commons.MeshMtlsNotSatisfied)
		}
	}

	return true, ""
}

func (v *MeshValidator) GetEntityType() client.Object {
	return &servicemeshapi.Mesh{}
}
