/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeployment

import (
	"context"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if len(vd.Spec.VirtualService.Id) == 0 {
		allowed, reason = validations.IsVSPresent(v.resolver, context, vd.Spec.VirtualService.ResourceRef, &vd.ObjectMeta)
		if !allowed {
			return false, reason
		}
	}

	allowed, reason = v.validateHostName(vd, context)
	if !allowed {
		return allowed, reason
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

	allowed, reason := v.validateHostName(vd, context)
	if !allowed {
		return allowed, reason
	}

	return true, ""
}

func (v *VirtualDeploymentValidator) validateServiceDiscovery(virtualDeployment *servicemeshapi.VirtualDeployment) (bool, string) {
	// TODO: Current implementation of MeshRegistry does not expect any fields.
	// If host name is needed for MeshRegistry, remove this and add a kube-validation instead

	if virtualDeployment.Spec.ServiceDiscovery == nil {
		return true, ""
	}

	hasHostname := len(virtualDeployment.Spec.ServiceDiscovery.Hostname) > 0
	if virtualDeployment.Spec.ServiceDiscovery.Type == servicemeshapi.ServiceDiscoveryTypeDns && !hasHostname {
		return false, string(commons.HostNameIsEmptyForDNS)
	}
	if virtualDeployment.Spec.ServiceDiscovery != nil && virtualDeployment.Spec.ServiceDiscovery.Type == servicemeshapi.ServiceDiscoveryTypeDisabled && hasHostname {
		return false, string(commons.HostNameShouldBeEmptyForDISABLED)
	}

	return true, ""
}

func (v *VirtualDeploymentValidator) validateHostName(virtualDeployment *servicemeshapi.VirtualDeployment, ctx context.Context) (bool, string) {
	hasServiceDiscovery := virtualDeployment.Spec.ServiceDiscovery != nil && virtualDeployment.Spec.ServiceDiscovery.Type != servicemeshapi.ServiceDiscoveryTypeDisabled
	hasHostname := hasServiceDiscovery && len(virtualDeployment.Spec.ServiceDiscovery.Hostname) > 0
	hasListener := virtualDeployment.Spec.Listener != nil && len(virtualDeployment.Spec.Listener) > 0

	if hasHostname && hasListener {
		if v.HasVSHost(ctx, &virtualDeployment.Spec.VirtualService, &virtualDeployment.ObjectMeta) {
			return true, ""
		}
		return false, string(commons.VirtualServiceHostNotFound)
	} else {
		isValid, reason := v.validateServiceDiscovery(virtualDeployment)
		if !isValid {
			return isValid, reason
		}
		if hasHostname || hasListener {
			return false, string(commons.VirtualDeploymentOnlyHaveHostnameOrListener)
		}
		return true, ""
	}
}

func (v *VirtualDeploymentValidator) GetEntityType() client.Object {
	return &servicemeshapi.VirtualDeployment{}
}

func (v *VirtualDeploymentValidator) HasVSHost(ctx context.Context, resource *servicemeshapi.RefOrId, objectMeta *metav1.ObjectMeta) bool {
	if len(resource.Id) == 0 {
		resourceRef := v.resolver.ResolveResourceRef(resource.ResourceRef, objectMeta)
		referredVirtualService, err := v.resolver.ResolveVirtualServiceReference(ctx, resourceRef)
		if err != nil {
			// this is the case where the referred VS in VD is not found or not active yet
			// and to support helm we should bypass this validation
			return true
		}
		return referredVirtualService.Spec.Hosts != nil && len(referredVirtualService.Spec.Hosts) > 0
	} else {
		virtualService, err := v.resolver.ResolveVirtualServiceById(ctx, &resource.Id)
		if err != nil {
			// this is the case where the referred VS in VD is not found or not active yet
			// and to support helm we should bypass this validation
			return true
		}
		return virtualService.Hosts != nil && len(virtualService.Hosts) > 0
	}
}
