/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package conversions

import (
	"fmt"
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

type IGRTDependencies struct {
	IngressGatewayId   api.OCID
	IngressGatewayName v1beta1.Name
	VsIdForRules       [][]api.OCID
}

func ConvertCrdIngressGatewayRouteTableToSdkIngressGatewayRouteTable(crdObj *v1beta1.IngressGatewayRouteTable, sdkObj *sdk.IngressGatewayRouteTable, dependencies *IGRTDependencies) error {
	sdkObj.Id = (*string)(&crdObj.Status.IngressGatewayRouteTableId)
	sdkObj.CompartmentId = (*string)(&crdObj.Spec.CompartmentId)
	sdkObj.Name = GetSpecName(crdObj.Spec.Name, &crdObj.ObjectMeta)
	sdkObj.Description = (*string)(crdObj.Spec.Description)
	sdkObj.IngressGatewayId = (*string)(&dependencies.IngressGatewayId)
	sdkObj.RouteRules = make([]sdk.IngressGatewayTrafficRouteRule, len(crdObj.Spec.RouteRules))
	for i := range crdObj.Spec.RouteRules {
		err := convertCrdIngressGatewayTrafficRouteRuleToSdkIngressGatewayTrafficRouteRule(&crdObj.Spec.RouteRules[i], &sdkObj.RouteRules[i], dependencies.VsIdForRules[i])
		if err != nil {
			return err
		}
	}
	sdkObj.Priority = crdObj.Spec.Priority
	if crdObj.Spec.FreeFormTags != nil {
		sdkObj.FreeformTags = crdObj.Spec.FreeFormTags
	}
	if crdObj.Spec.DefinedTags != nil {
		sdkObj.DefinedTags = map[string]map[string]interface{}{}
		ConvertCrdDefinedTagsToSdkDefinedTags(&crdObj.Spec.DefinedTags, &sdkObj.DefinedTags)
	}
	return nil
}

func convertCrdIngressGatewayTrafficRouteRuleToSdkIngressGatewayTrafficRouteRule(crdObj *v1beta1.IngressGatewayTrafficRouteRule, sdkObj *sdk.IngressGatewayTrafficRouteRule,
	vsIds []api.OCID) error {
	if route := crdObj.HttpRoute; route != nil {
		destinations, err := convertToSdkDestinations(crdObj, vsIds)
		if err != nil {
			return err
		}
		*sdkObj = sdk.HttpIngressGatewayTrafficRouteRule{
			Destinations:         destinations,
			IngressGatewayHost:   convertCrdIngressGatewayHostToSdkIngressGatewayHost(route.IngressGatewayHost),
			Path:                 route.Path,
			IsGrpc:               route.IsGrpc,
			PathType:             sdk.HttpIngressGatewayTrafficRouteRulePathTypeEnum(route.PathType),
			IsHostRewriteEnabled: route.IsHostRewriteEnabled,
			IsPathRewriteEnabled: route.IsPathRewriteEnabled,
		}
	} else if route := crdObj.TcpRoute; route != nil {
		destinations, err := convertToSdkDestinations(crdObj, vsIds)
		if err != nil {
			return err
		}
		*sdkObj = sdk.TcpIngressGatewayTrafficRouteRule{
			Destinations:       destinations,
			IngressGatewayHost: convertCrdIngressGatewayHostToSdkIngressGatewayHost(route.IngressGatewayHost),
		}
	} else if route := crdObj.TlsPassthroughRoute; route != nil {
		destinations, err := convertToSdkDestinations(crdObj, vsIds)
		if err != nil {
			return err
		}
		*sdkObj = sdk.TlsPassthroughIngressGatewayTrafficRouteRule{
			Destinations:       destinations,
			IngressGatewayHost: convertCrdIngressGatewayHostToSdkIngressGatewayHost(route.IngressGatewayHost),
		}
	} else {
		return fmt.Errorf("missing ingress gateway route in %v", crdObj)
	}
	return nil
}

func convertToSdkDestinations(route *v1beta1.IngressGatewayTrafficRouteRule, vsIds []api.OCID) ([]sdk.VirtualServiceTrafficRuleTarget, error) {
	destinations, err := getVirtualServiceDestinations(route)
	if err != nil {
		return nil, err
	}
	sdkObjDestinations := make([]sdk.VirtualServiceTrafficRuleTarget, len(destinations))
	for i := range destinations {
		ConvertCrdTrafficRouteRuleDestinationToSdkTrafficRouteRuleDestinationForIGRT(&destinations[i], &sdkObjDestinations[i], vsIds[i])
	}
	return sdkObjDestinations, nil
}

func getVirtualServiceDestinations(rule *v1beta1.IngressGatewayTrafficRouteRule) ([]v1beta1.VirtualServiceTrafficRuleTarget, error) {
	switch {
	case rule.HttpRoute != nil:
		return rule.HttpRoute.Destinations, nil
	case rule.TcpRoute != nil:
		return rule.TcpRoute.Destinations, nil
	case rule.TlsPassthroughRoute != nil:
		return rule.TlsPassthroughRoute.Destinations, nil
	default:
		return nil, fmt.Errorf("missing destinations in route rule %v", rule)
	}
}

func ConvertCrdTrafficRouteRuleDestinationToSdkTrafficRouteRuleDestinationForIGRT(crdObj *v1beta1.VirtualServiceTrafficRuleTarget, sdkObj *sdk.VirtualServiceTrafficRuleTarget, vsID api.OCID) {
	sdkObj.VirtualServiceId = (*string)(&vsID)
	sdkObj.Weight = crdObj.Weight
	sdkObj.Port = PortToInt(crdObj.Port)
}

func convertCrdIngressGatewayHostToSdkIngressGatewayHost(crdIngressGatewayHost *v1beta1.IngressGatewayHostRef) *sdk.IngressGatewayHostRef {
	if crdIngressGatewayHost == nil {
		return nil
	}
	sdkIngressGatewayHostRef := sdk.IngressGatewayHostRef{Name: (*string)(&crdIngressGatewayHost.Name),
		Port: PortToInt(crdIngressGatewayHost.Port),
	}
	return &sdkIngressGatewayHostRef
}
