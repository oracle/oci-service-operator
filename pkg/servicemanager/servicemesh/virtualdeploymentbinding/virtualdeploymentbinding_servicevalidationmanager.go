/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualdeploymentbinding

import (
	"context"
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/validations"
)

//+kubebuilder:webhook:path=/validate-servicemesh-oci-oracle-com-v1beta1-virtualdeploymentbinding,mutating=false,failurePolicy=fail,sideEffects=None,groups=servicemesh.oci.oracle.com,resources=virtualdeploymentbindings,verbs=create;update;delete,versions=v1beta1,name=vdb-validator.servicemesh.oci.oracle.cloud.com,admissionReviewVersions={v1,v1beta1}

type VirtualDeploymentBindingValidator struct {
	resolver references.Resolver
	log      loggerutil.OSOKLogger
}

func NewVirtualDeploymentBindingValidator(resolver references.Resolver, log loggerutil.OSOKLogger) manager.CustomResourceValidator {
	return &VirtualDeploymentBindingValidator{resolver: resolver, log: log}
}

func (v *VirtualDeploymentBindingValidator) ValidateOnCreate(context context.Context, object client.Object) (bool, string) {
	vdb, err := getVirtualDeploymentBinding(object)
	if err != nil {
		return false, err.Error()
	}

	allowed, reason := validations.IsMetadataNameValid(vdb.GetName())
	if !allowed {
		return false, reason
	}

	// Only validate for the requests come from k8s operator
	if len(vdb.Spec.VirtualDeployment.Id) == 0 {
		return v.isReferredResourceExist(context, vdb)
	}

	return true, ""
}

func (v *VirtualDeploymentBindingValidator) GetStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	vdb, err := getVirtualDeploymentBinding(object)
	if err != nil {
		return nil, err
	}
	return &vdb.Status, nil
}

func (v *VirtualDeploymentBindingValidator) ResolveRef(object client.Object) (bool, string) {
	vdb, err := getVirtualDeploymentBinding(object)
	if err != nil {
		return false, err.Error()
	}
	err = validations.ValidateVDRef(&vdb.Spec.VirtualDeployment)
	if err != nil {
		return false, err.Error()
	}
	return true, ""
}

func (v *VirtualDeploymentBindingValidator) ValidateObject(object client.Object) error {
	_, err := getVirtualDeploymentBinding(object)
	return err
}

func (v *VirtualDeploymentBindingValidator) ValidateOnUpdate(context context.Context, object client.Object, oldObject client.Object) (bool, string) {
	vdb, err := getVirtualDeploymentBinding(object)
	if err != nil {
		return false, err.Error()
	}
	err = validations.ValidateVDRef(&vdb.Spec.VirtualDeployment)
	if err != nil {
		return false, err.Error()
	}
	// Only validate for the requests come from k8s operator
	if len(vdb.Spec.VirtualDeployment.Id) == 0 {
		return v.isReferredResourceExist(context, vdb)
	}
	return true, ""
}

func getVirtualDeploymentBinding(object client.Object) (*servicemeshapi.VirtualDeploymentBinding, error) {
	vdb, ok := object.(*servicemeshapi.VirtualDeploymentBinding)
	if !ok {
		return nil, errors.New("object is not a virtual deployment binding")
	}
	return vdb, nil
}

func (v *VirtualDeploymentBindingValidator) isReferredResourceExist(ctx context.Context, binding *servicemeshapi.VirtualDeploymentBinding) (bool, string) {
	allowed, reason := validations.IsVDPresent(v.resolver, ctx, binding.Spec.VirtualDeployment.ResourceRef, &binding.ObjectMeta)
	if !allowed {
		return allowed, reason
	}
	allowed, reason = validations.IsServicePresent(v.resolver, ctx, &binding.Spec.Target.Service.ServiceRef, &binding.ObjectMeta)
	if !allowed {
		return allowed, reason
	}

	return true, ""
}

func (v *VirtualDeploymentBindingValidator) GetEntityType() client.Object {
	return &servicemeshapi.VirtualDeploymentBinding{}
}
