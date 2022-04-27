/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package accesspolicy

import (
	"context"
	"errors"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/controller-runtime/pkg/client"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/validations"
)

//+kubebuilder:webhook:path=/validate-servicemesh-oci-oracle-com-v1beta1-accesspolicy,mutating=false,failurePolicy=fail,sideEffects=None,groups=servicemesh.oci.oracle.com,resources=accesspolicies,verbs=create;update;delete,versions=v1beta1,name=ap-validator.servicemesh.oci.oracle.cloud.com,admissionReviewVersions={v1,v1beta1}

type AccessPolicyValidator struct {
	resolver references.Resolver
	log      loggerutil.OSOKLogger
}

func NewAccessPolicyValidator(resolver references.Resolver, log loggerutil.OSOKLogger) manager.CustomResourceValidator {
	return &AccessPolicyValidator{resolver: resolver, log: log}
}

func (v *AccessPolicyValidator) ValidateOnCreate(context context.Context, object client.Object) (bool, string) {
	ap, err := getAccessPolicy(object)
	if err != nil {
		return false, err.Error()
	}

	allowed, reason := validations.IsMetadataNameValid(ap.GetName())
	if !allowed {
		return false, reason
	}

	allowed, reason = v.validateAccessPolicyTargets(ap)
	if !allowed {
		return false, reason
	}

	// Only validate for the requests come from k8s operator
	if len(ap.Spec.Mesh.Id) == 0 {
		allowed, reason = validations.IsMeshPresent(v.resolver, context, ap.Spec.Mesh.ResourceRef, &ap.ObjectMeta)
		if !allowed {
			return false, reason
		}
	}

	// Check if the requests that comes from k8s operator refer to valid virtual service
	allowed, reason = v.hasValidResourceInAccessPolicy(context, ap)
	if !allowed {
		return false, reason
	}
	return true, ""
}

func (v *AccessPolicyValidator) GetStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	ap, err := getAccessPolicy(object)
	if err != nil {
		return nil, err
	}
	return &ap.Status, nil
}

func (v *AccessPolicyValidator) ResolveRef(object client.Object) (bool, string) {
	ap, err := getAccessPolicy(object)
	if err != nil {
		return false, err.Error()
	}
	if err := validations.ValidateMeshRef(&ap.Spec.Mesh); err != nil {
		return false, err.Error()
	}
	return v.hasOCIDOrK8sReferenceInAccessPolicyTrafficTargets(ap)
}

func (v *AccessPolicyValidator) ValidateObject(object client.Object) error {
	_, err := getAccessPolicy(object)
	return err
}

func (v *AccessPolicyValidator) ValidateOnUpdate(context context.Context, object client.Object, oldObject client.Object) (bool, string) {
	ap, err := getAccessPolicy(object)
	if err != nil {
		return false, err.Error()
	}
	oldAp, err := getAccessPolicy(oldObject)
	if err != nil {
		return false, err.Error()
	}

	if !cmp.Equal(ap.Spec.Mesh, oldAp.Spec.Mesh) {
		return false, string(commons.MeshReferenceIsImmutable)
	}

	// Throw an error if name has changed
	if validations.IsSpecNameChanged(ap.Spec.Name, oldAp.Spec.Name) {
		return false, string(commons.NameIsImmutable)
	}

	allowed, reason := v.validateAccessPolicyTargets(ap)
	if !allowed {
		return false, reason
	}

	allowed, reason = v.hasOCIDOrK8sReferenceInAccessPolicyTrafficTargets(ap)
	if !allowed {
		return false, reason
	}

	allowed, reason = v.hasValidResourceInAccessPolicy(context, ap)
	if !allowed {
		return false, reason
	}

	return true, ""
}

func (v *AccessPolicyValidator) hasOCIDOrK8sReferenceInAccessPolicyTrafficTargets(accessPolicy *servicemeshapi.AccessPolicy) (bool, string) {
	for _, accessPolicyRule := range accessPolicy.Spec.Rules {
		source := accessPolicyRule.Source
		destination := accessPolicyRule.Destination
		if source.VirtualService != nil {
			err := validations.ValidateVSRef(source.VirtualService)
			if err != nil {
				return false, err.Error()
			}
		} else if source.IngressGateway != nil {
			err := validations.ValidateIGRef(source.IngressGateway)
			if err != nil {
				return false, err.Error()
			}
		}
		if destination.VirtualService != nil {
			err := validations.ValidateVSRef(destination.VirtualService)
			if err != nil {
				return false, err.Error()
			}
		}
	}
	return true, ""
}

func (v *AccessPolicyValidator) validateAccessPolicyTargets(accessPolicy *servicemeshapi.AccessPolicy) (bool, string) {
	for _, accessPolicyRule := range accessPolicy.Spec.Rules {
		if err := v.validateSourceTargetType(accessPolicyRule.Source); err != nil {
			return false, err.Error()
		}
		if err := v.validateDestinationTargetType(accessPolicyRule.Destination); err != nil {
			return false, err.Error()
		}
	}
	return true, ""
}

func (v *AccessPolicyValidator) validateSourceTargetType(source servicemeshapi.TrafficTarget) error {
	if err := validateOneOfAccessPolicyTarget(source); err != nil {
		return err
	}
	if source.ExternalService != nil {
		return errors.New("invalid source access policy target. source should be one of: allVirtualServices; virtualService; ingressGateway")
	}
	return nil
}

func (v *AccessPolicyValidator) validateDestinationTargetType(destination servicemeshapi.TrafficTarget) error {
	if err := validateOneOfAccessPolicyTarget(destination); err != nil {
		return err
	}
	if destination.IngressGateway != nil {
		return errors.New("invalid destination access policy target. destination should be one of: allVirtualServices; virtualService; externalService")
	}
	if destination.ExternalService != nil {
		if err := validateExternalServiceTarget(destination.ExternalService); err != nil {
			return err
		}
	}
	return nil
}

func validateExternalServiceTarget(service *servicemeshapi.ExternalService) error {
	if service == nil {
		return errors.New("missing external service target")
	}
	externalServiceTypeCount := 0
	if service.HttpExternalService != nil {
		externalServiceTypeCount++
	}
	if service.HttpsExternalService != nil {
		externalServiceTypeCount++
	}
	if service.TcpExternalService != nil {
		externalServiceTypeCount++
	}

	switch {
	case externalServiceTypeCount == 0:
		return errors.New("missing external service target")
	case externalServiceTypeCount > 1:
		return errors.New("cannot specify more than one external service type")
	}

	return nil
}

func validateOneOfAccessPolicyTarget(trafficTarget servicemeshapi.TrafficTarget) error {
	targetTypeCount := 0
	if trafficTarget.ExternalService != nil {
		targetTypeCount++
	}
	if trafficTarget.AllVirtualServices != nil {
		targetTypeCount++
	}
	if trafficTarget.VirtualService != nil {
		targetTypeCount++
	}
	if trafficTarget.IngressGateway != nil {
		targetTypeCount++
	}

	switch {
	case targetTypeCount == 0:
		return errors.New("access policy target cannot be empty")
	case targetTypeCount > 1:
		return errors.New("access policy target cannot contain more than one type")
	}

	return nil
}

func (v *AccessPolicyValidator) hasValidResourceInAccessPolicy(ctx context.Context, accessPolicy *servicemeshapi.AccessPolicy) (bool, string) {
	for _, accessPolicyRule := range accessPolicy.Spec.Rules {
		source := accessPolicyRule.Source
		destination := accessPolicyRule.Destination
		if source.VirtualService != nil && len(source.VirtualService.Id) == 0 {
			allowed, reason := validations.IsVSPresent(v.resolver, ctx, source.VirtualService.ResourceRef, &accessPolicy.ObjectMeta)
			if !allowed {
				return allowed, reason
			}
		}
		if destination.VirtualService != nil && len(destination.VirtualService.Id) == 0 {
			allowed, reason := validations.IsVSPresent(v.resolver, ctx, destination.VirtualService.ResourceRef, &accessPolicy.ObjectMeta)
			if !allowed {
				return allowed, reason
			}
		}
	}
	return true, ""
}

func (v *AccessPolicyValidator) GetEntityType() client.Object {
	return &servicemeshapi.AccessPolicy{}
}
