/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeployment

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

//+kubebuilder:webhook:path=/validate-servicemesh-oci-oracle-com-v1beta1-virtualdeployment,mutating=false,failurePolicy=fail,sideEffects=None,groups=servicemesh.oci.oracle.com,resources=virtualdeployments,verbs=create;update;delete,versions=v1beta1,name=vd-validator.servicemesh.oci.oracle.cloud.com,admissionReviewVersions={v1,v1beta1}

type VirtualDeploymentValidator struct {
	resolver references.Resolver
	log      loggerutil.OSOKLogger
}

func NewVirtualDeploymentValidator(resolver references.Resolver, log loggerutil.OSOKLogger) manager.CustomResourceValidator {
	return &VirtualDeploymentValidator{resolver: resolver, log: log}
}

func (v *VirtualDeploymentValidator) ValidateOnCreate(context context.Context, object client.Object) (bool, string) {
	vd, err := getVirtualDeployment(object)
	if err != nil {
		return false, err.Error()
	}

	allowed, reason := validations.IsMetadataNameValid(vd.GetName())
	if !allowed {
		return false, reason
	}

	if !v.validateHostName(vd) {
		return false, string(commons.HostNameIsEmptyForDNS)
	}

	if len(vd.Spec.VirtualService.Id) == 0 {
		allowed, reason = validations.IsVSPresent(v.resolver, context, vd.Spec.VirtualService.ResourceRef, &vd.ObjectMeta)
		if !allowed {
			return false, reason
		}
	}

	return true, ""
}

func (v *VirtualDeploymentValidator) GetStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	vd, err := getVirtualDeployment(object)
	if err != nil {
		return nil, err
	}
	return &vd.Status, nil
}

func (v *VirtualDeploymentValidator) ResolveRef(object client.Object) (bool, string) {
	vd, err := getVirtualDeployment(object)
	if err != nil {
		return false, err.Error()
	}
	err = validations.ValidateVSRef(&vd.Spec.VirtualService)
	if err != nil {
		return false, err.Error()
	}
	return true, ""
}

func (v *VirtualDeploymentValidator) ValidateObject(object client.Object) error {
	_, err := getVirtualDeployment(object)
	return err
}

func (v *VirtualDeploymentValidator) ValidateOnUpdate(context context.Context, object client.Object, oldObject client.Object) (bool, string) {
	vd, err := getVirtualDeployment(object)
	if err != nil {
		return false, err.Error()
	}
	oldVd, err := getVirtualDeployment(oldObject)
	if err != nil {
		return false, err.Error()
	}

	if !cmp.Equal(vd.Spec.VirtualService, oldVd.Spec.VirtualService) {
		return false, string(commons.VirtualServiceReferenceIsImmutable)
	}

	// Throw an error if name has changed
	if validations.IsSpecNameChanged(vd.Spec.Name, oldVd.Spec.Name) {
		return false, string(commons.NameIsImmutable)
	}

	if !v.validateHostName(vd) {
		return false, string(commons.HostNameIsEmptyForDNS)
	}

	return true, ""
}

func (v *VirtualDeploymentValidator) validateHostName(virtualDeployment *servicemeshapi.VirtualDeployment) bool {
	// TODO: Current implementation of MeshRegistry does not expect any fields.
	// If host name is needed for MeshRegistry, remove this and add a kube-validation instead
	if virtualDeployment.Spec.ServiceDiscovery.Type == servicemeshapi.ServiceDiscoveryTypeDns && len(virtualDeployment.Spec.ServiceDiscovery.Hostname) == 0 {
		return false
	}
	return true
}

func (v *VirtualDeploymentValidator) GetEntityType() client.Object {
	return &servicemeshapi.VirtualDeployment{}
}
