/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgatewayroutetable

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

//+kubebuilder:webhook:path=/validate-servicemesh-oci-oracle-com-v1beta1-ingressgatewayroutetable,mutating=false,failurePolicy=fail,sideEffects=None,groups=servicemesh.oci.oracle.com,resources=ingressgatewayroutetables,verbs=create;update;delete,versions=v1beta1,name=igrt-validator.servicemesh.oci.oracle.cloud.com,admissionReviewVersions={v1,v1beta1}

type IngressGatewayRouteTableValidator struct {
	resolver references.Resolver
	log      loggerutil.OSOKLogger
}

func NewIngressGatewayRouteTableValidator(resolver references.Resolver, log loggerutil.OSOKLogger) manager.CustomResourceValidator {
	return &IngressGatewayRouteTableValidator{resolver: resolver, log: log}
}

func (v *IngressGatewayRouteTableValidator) ValidateOnCreate(context context.Context, object client.Object) (bool, string) {
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return false, err.Error()
	}

	allowed, reason := validations.IsMetadataNameValid(igrt.GetName())
	if !allowed {
		return false, reason
	}

	allowed, reason = v.validateTrafficRouteRuleInIngressGatewayRouteTable(igrt)
	if !allowed {
		return false, reason
	}

	if len(igrt.Spec.IngressGateway.Id) == 0 {
		allowed, reason = validations.IsIGPresent(v.resolver, context, igrt.Spec.IngressGateway.ResourceRef, &igrt.ObjectMeta)
		if !allowed {
			return false, reason
		}

	}

	// Check if the requests that comes from k8s operator refer to valid virtual services
	allowed, reason = v.hasValidResourceInRouteTable(context, igrt)
	if !allowed {
		return false, reason
	}

	return true, ""
}

func (v *IngressGatewayRouteTableValidator) GetStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return nil, err
	}
	return &igrt.Status, nil
}

func (v *IngressGatewayRouteTableValidator) ResolveRef(object client.Object) (bool, string) {
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return false, err.Error()
	}
	err = validations.ValidateIGRef(&igrt.Spec.IngressGateway)
	if err != nil {
		return false, err.Error()
	}
	return v.hasOCIDOrK8sReferenceInIngressGatewayRouteTableDestinations(igrt)
}

func (v *IngressGatewayRouteTableValidator) ValidateObject(object client.Object) error {
	_, err := getIngressGatewayRouteTable(object)
	return err
}

func (v *IngressGatewayRouteTableValidator) ValidateOnUpdate(context context.Context, object client.Object, oldObject client.Object) (bool, string) {
	igrt, err := getIngressGatewayRouteTable(object)
	if err != nil {
		return false, err.Error()
	}
	oldIgrt, err := getIngressGatewayRouteTable(oldObject)
	if err != nil {
		return false, err.Error()
	}

	if !cmp.Equal(igrt.Spec.IngressGateway, oldIgrt.Spec.IngressGateway) {
		return false, string(commons.IngressGatewayReferenceIsImmutable)
	}

	// Throw an error if name has changed
	if validations.IsSpecNameChanged(igrt.Spec.Name, oldIgrt.Spec.Name) {
		return false, string(commons.NameIsImmutable)
	}

	allowed, reason := v.validateTrafficRouteRuleInIngressGatewayRouteTable(igrt)
	if !allowed {
		return false, reason
	}

	allowed, reason = v.hasOCIDOrK8sReferenceInIngressGatewayRouteTableDestinations(igrt)
	if !allowed {
		return false, reason
	}

	allowed, reason = v.hasValidResourceInRouteTable(context, igrt)
	if !allowed {
		return false, reason
	}

	return true, ""
}

func (v *IngressGatewayRouteTableValidator) hasOCIDOrK8sReferenceInIngressGatewayRouteTableDestinations(ingressGatewayRouteTable *servicemeshapi.IngressGatewayRouteTable) (bool, string) {
	for _, routeRule := range ingressGatewayRouteTable.Spec.RouteRules {
		destinations, err := getVirtualServiceDestinations(routeRule)
		if err != nil {
			return false, err.Error()
		}
		for _, destination := range destinations {
			err := validations.ValidateVSRef(destination.VirtualService)
			if err != nil {
				return false, err.Error()
			}
		}
	}
	return true, ""
}

func (v *IngressGatewayRouteTableValidator) validateTrafficRouteRuleInIngressGatewayRouteTable(ingressGatewayRouteTable *servicemeshapi.IngressGatewayRouteTable) (bool, string) {
	for _, routeRule := range ingressGatewayRouteTable.Spec.RouteRules {
		if err := validateTrafficRouteRuleInIngressGatewayRouteTable(&routeRule); err != nil {
			return false, err.Error()
		}
	}
	return true, ""
}

func validateTrafficRouteRuleInIngressGatewayRouteTable(routeRule *servicemeshapi.IngressGatewayTrafficRouteRule) error {
	if routeRule == nil {
		return errors.New("missing ingress gateway route rule")
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

func (v *IngressGatewayRouteTableValidator) hasValidResourceInRouteTable(ctx context.Context, ingressGatewayRouteTable *servicemeshapi.IngressGatewayRouteTable) (bool, string) {
	for _, routeRule := range ingressGatewayRouteTable.Spec.RouteRules {
		destinations, err := getVirtualServiceDestinations(routeRule)
		if err != nil {
			return false, err.Error()
		}
		for _, destination := range destinations {
			if len(destination.VirtualService.Id) == 0 {
				allowed, reason := validations.IsVSPresent(v.resolver, ctx, destination.VirtualService.ResourceRef, &ingressGatewayRouteTable.ObjectMeta)
				if !allowed {
					return allowed, reason
				}
			}
		}
	}
	return true, ""
}

func (v *IngressGatewayRouteTableValidator) GetEntityType() client.Object {
	return &servicemeshapi.IngressGatewayRouteTable{}
}
