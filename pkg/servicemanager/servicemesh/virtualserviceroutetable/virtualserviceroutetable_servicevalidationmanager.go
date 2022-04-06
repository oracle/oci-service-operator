/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package virtualserviceroutetable

import (
	"context"
	"errors"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/controller-runtime/pkg/client"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/validations"
)

//+kubebuilder:webhook:path=/validate-servicemesh-oci-oracle-com-v1beta1-virtualserviceroutetable,mutating=false,failurePolicy=fail,sideEffects=None,groups=servicemesh.oci.oracle.com,resources=virtualserviceroutetables,verbs=create;update;delete,versions=v1beta1,name=vsrt-validator.servicemesh.oci.oracle.cloud.com,admissionReviewVersions={v1,v1beta1}

type VirtualServiceRouteTableValidator struct {
	resolver references.Resolver
	log      loggerutil.OSOKLogger
}

func NewVirtualServiceRouteTableValidator(resolver references.Resolver, log loggerutil.OSOKLogger) manager.CustomResourceValidator {
	return &VirtualServiceRouteTableValidator{resolver: resolver, log: log}
}

func (v *VirtualServiceRouteTableValidator) ValidateOnCreate(context context.Context, object client.Object) (bool, string) {
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return false, err.Error()
	}

	allowed, reason := validations.IsMetadataNameValid(vsrt.GetName())
	if !allowed {
		return false, reason
	}

	allowed, reason = v.validateTrafficRouteRuleInVirtualServiceRouteTable(vsrt)
	if !allowed {
		return false, reason
	}

	if len(vsrt.Spec.VirtualService.Id) == 0 {
		allowed, reason = validations.IsVSPresent(v.resolver, context, vsrt.Spec.VirtualService.ResourceRef, &vsrt.ObjectMeta)
		if !allowed {
			return false, reason
		}

	}

	// Check if the requests that comes from k8s operator refer to valid virtual deployments
	allowed, reason = v.hasValidResourceInRouteTable(context, vsrt)
	if !allowed {
		return false, reason
	}

	return true, ""
}

func (v *VirtualServiceRouteTableValidator) GetStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return nil, err
	}
	return &vsrt.Status, nil
}

func (v *VirtualServiceRouteTableValidator) ResolveRef(object client.Object) (bool, string) {
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return false, err.Error()
	}
	err = validations.ValidateVSRef(&vsrt.Spec.VirtualService)
	if err != nil {
		return false, err.Error()
	}
	return v.hasOCIDOrK8sReferenceInVirtualServiceRouteTableDestinations(vsrt)
}

func (v *VirtualServiceRouteTableValidator) ValidateObject(object client.Object) error {
	_, err := getVirtualServiceRouteTable(object)
	return err
}

func (v *VirtualServiceRouteTableValidator) ValidateOnUpdate(context context.Context, object client.Object, oldObject client.Object) (bool, string) {
	vsrt, err := getVirtualServiceRouteTable(object)
	if err != nil {
		return false, err.Error()
	}
	oldVsrt, err := getVirtualServiceRouteTable(oldObject)
	if err != nil {
		return false, err.Error()
	}

	if !cmp.Equal(vsrt.Spec.VirtualService, oldVsrt.Spec.VirtualService) {
		return false, string(commons.VirtualServiceReferenceIsImmutable)
	}

	// Throw an error if name has changed
	if validations.IsSpecNameChanged(vsrt.Spec.Name, oldVsrt.Spec.Name) {
		return false, string(commons.NameIsImmutable)
	}

	allowed, reason := v.validateTrafficRouteRuleInVirtualServiceRouteTable(vsrt)
	if !allowed {
		return false, reason
	}

	allowed, reason = v.hasOCIDOrK8sReferenceInVirtualServiceRouteTableDestinations(vsrt)
	if !allowed {
		return false, reason
	}

	allowed, reason = v.hasValidResourceInRouteTable(context, vsrt)
	if !allowed {
		return false, reason
	}

	return true, ""
}

func (v *VirtualServiceRouteTableValidator) validateTrafficRouteRuleInVirtualServiceRouteTable(virtualServiceRouteTable *servicemeshapi.VirtualServiceRouteTable) (bool, string) {
	for _, routeRule := range virtualServiceRouteTable.Spec.RouteRules {
		if err := validateTrafficRouteRuleOneOff(&routeRule); err != nil {
			return false, err.Error()
		}
		if err := validateRouteRuleDestinations(routeRule); err != nil {
			return false, err.Error()
		}
	}
	return true, ""
}

func validateRouteRuleDestinations(routeRule servicemeshapi.VirtualServiceTrafficRouteRule) error {
	destinations, err := getVirtualDeploymentDestinations(routeRule)
	if err != nil {
		return err
	}
	portMap := map[int]bool{}
	for _, destination := range destinations {
		port := conversions.PortToInt(destination.Port)
		if port != nil {
			portMap[*port] = true
		} else {
			portMap[0] = true
		}
	}
	if len(portMap) != 1 {
		return errors.New("route rule destinations cannot have different ports")
	}
	return nil
}

func validateTrafficRouteRuleOneOff(routeRule *servicemeshapi.VirtualServiceTrafficRouteRule) error {
	if routeRule == nil {
		return errors.New("missing virtual service route rule")
	}
	ruleTypeCount := 0
	if routeRule.HttpRoute != nil {
		ruleTypeCount++
	}
	if routeRule.TcpRoute != nil {
		ruleTypeCount++
	}
	if routeRule.TlsPassthroughRoute != nil {
		ruleTypeCount++
	}

	switch {
	case ruleTypeCount == 0:
		return errors.New(string(commons.TrafficRouteRuleIsEmpty))
	case ruleTypeCount > 1:
		return errors.New(string(commons.TrafficRouteRuleIsNotUnique))
	}

	return nil
}

func (v *VirtualServiceRouteTableValidator) hasValidResourceInRouteTable(ctx context.Context, virtualServiceRouteTable *servicemeshapi.VirtualServiceRouteTable) (bool, string) {
	for _, routeRule := range virtualServiceRouteTable.Spec.RouteRules {
		destinations, err := getVirtualDeploymentDestinations(routeRule)
		if err != nil {
			return false, err.Error()
		}
		for _, destination := range destinations {
			if len(destination.VirtualDeployment.Id) == 0 {
				allowed, reason := validations.IsVDPresent(v.resolver, ctx, destination.VirtualDeployment.ResourceRef, &virtualServiceRouteTable.ObjectMeta)
				if !allowed {
					return allowed, reason
				}
			}
		}
	}
	return true, ""
}

func (v *VirtualServiceRouteTableValidator) hasOCIDOrK8sReferenceInVirtualServiceRouteTableDestinations(vsrt *servicemeshapi.VirtualServiceRouteTable) (bool, string) {
	for _, routeRule := range vsrt.Spec.RouteRules {
		destinations, err := getVirtualDeploymentDestinations(routeRule)
		if err != nil {
			return false, err.Error()
		}
		for _, destination := range destinations {
			err := validations.ValidateVDRef(destination.VirtualDeployment)
			if err != nil {
				return false, err.Error()
			}
		}
	}
	return true, ""
}

func (v *VirtualServiceRouteTableValidator) GetEntityType() client.Object {
	return &servicemeshapi.VirtualServiceRouteTable{}
}
