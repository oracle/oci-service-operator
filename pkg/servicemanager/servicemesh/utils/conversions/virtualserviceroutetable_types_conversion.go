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

type VSRTDependencies struct {
	VirtualServiceId   api.OCID
	VirtualServiceName v1beta1.Name
	VdIdForRules       [][]api.OCID
}

func ConvertCrdVsrtToSdkVsrt(crdObj *v1beta1.VirtualServiceRouteTable, sdkObj *sdk.VirtualServiceRouteTable, dependencies *VSRTDependencies) error {
	sdkObj.VirtualServiceId = (*string)(&dependencies.VirtualServiceId)
	sdkObj.Id = (*string)(&crdObj.Status.VirtualServiceRouteTableId)
	sdkObj.CompartmentId = (*string)(&crdObj.Spec.CompartmentId)
	sdkObj.Name = GetSpecName(crdObj.Spec.Name, &crdObj.ObjectMeta)
	sdkObj.Description = (*string)(crdObj.Spec.Description)
	sdkObj.RouteRules = make([]sdk.VirtualServiceTrafficRouteRule, len(crdObj.Spec.RouteRules))
	for i := range crdObj.Spec.RouteRules {
		err := ConvertCrdVsrtTrafficRouteRuleToSdkTrafficRouteRule(&crdObj.Spec.RouteRules[i], &sdkObj.RouteRules[i], dependencies.VdIdForRules[i])
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

func ConvertCrdVsrtTrafficRouteRuleToSdkTrafficRouteRule(crdObj *v1beta1.VirtualServiceTrafficRouteRule, sdkObj *sdk.VirtualServiceTrafficRouteRule, vdIds []api.OCID) error {
	if route := crdObj.HttpRoute; route != nil {
		sdkObjDestinations, err := convertToSdkVirtualDeploymentDestinations(crdObj, vdIds)
		if err != nil {
			return err
		}
		*sdkObj = sdk.HttpVirtualServiceTrafficRouteRule{
			Destinations: sdkObjDestinations,
			Path:         route.Path,
			IsGrpc:       route.IsGrpc,
			PathType:     sdk.HttpVirtualServiceTrafficRouteRulePathTypeEnum(route.PathType),
		}
	} else if route := crdObj.TcpRoute; route != nil {
		sdkObjDestinations, err := convertToSdkVirtualDeploymentDestinations(crdObj, vdIds)
		if err != nil {
			return err
		}
		*sdkObj = sdk.TcpVirtualServiceTrafficRouteRule{
			Destinations: sdkObjDestinations,
		}
	} else if route := crdObj.TlsPassthroughRoute; route != nil {
		sdkObjDestinations, err := convertToSdkVirtualDeploymentDestinations(crdObj, vdIds)
		if err != nil {
			return err
		}
		*sdkObj = sdk.TlsPassthroughVirtualServiceTrafficRouteRule{
			Destinations: sdkObjDestinations,
		}
	} else {
		return fmt.Errorf("missing virtual service route in %v", crdObj)
	}
	return nil
}

func convertToSdkVirtualDeploymentDestinations(route *v1beta1.VirtualServiceTrafficRouteRule, vdIds []api.OCID) ([]sdk.VirtualDeploymentTrafficRuleTarget, error) {
	destinations, err := getVirtualDeploymentDestinations(route)
	if err != nil {
		return nil, err
	}
	sdkObjDestinations := make([]sdk.VirtualDeploymentTrafficRuleTarget, len(destinations))
	for i := range destinations {
		ConvertCrdTrafficRouteRuleDestinationToSdkTrafficRouteRuleDestination(&destinations[i], &sdkObjDestinations[i], vdIds[i])
	}
	return sdkObjDestinations, nil
}

func getVirtualDeploymentDestinations(rule *v1beta1.VirtualServiceTrafficRouteRule) ([]v1beta1.VirtualDeploymentTrafficRuleTarget, error) {
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

func ConvertCrdTrafficRouteRuleDestinationToSdkTrafficRouteRuleDestination(crdObj *v1beta1.VirtualDeploymentTrafficRuleTarget, sdkObj *sdk.VirtualDeploymentTrafficRuleTarget, vdID api.OCID) {
	sdkObj.VirtualDeploymentId = (*string)(&vdID)
	sdkObj.Weight = &crdObj.Weight
	sdkObj.Port = PortToInt(crdObj.Port)
}
