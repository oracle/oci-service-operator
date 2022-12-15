/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package conversions

import (
	"errors"
	"fmt"

	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	meshCommons "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
)

type AccessPolicyDependencies struct {
	MeshId        api.OCID
	RefIdForRules []map[string]api.OCID
}

func ConvertCrdAccessPolicyToSdkAccessPolicy(crdObj *v1beta1.AccessPolicy, sdkObj *sdk.AccessPolicy, dependencies *AccessPolicyDependencies) {
	sdkObj.Id = (*string)(&crdObj.Status.AccessPolicyId)
	sdkObj.CompartmentId = (*string)(&crdObj.Spec.CompartmentId)
	sdkObj.Name = GetSpecName(crdObj.Spec.Name, &crdObj.ObjectMeta)
	sdkObj.Description = (*string)(crdObj.Spec.Description)
	sdkObj.MeshId = (*string)(&dependencies.MeshId)
	if crdObj.Spec.Rules != nil {
		sdkObj.Rules = make([]sdk.AccessPolicyRule, len(crdObj.Spec.Rules))
		for i := range crdObj.Spec.Rules {
			ConvertCrdAccessPolicyRuleToSdkAccessPolicyRule(&crdObj.Spec.Rules[i], &sdkObj.Rules[i], dependencies.RefIdForRules[i])
		}
	}
	if crdObj.Spec.FreeFormTags != nil {
		ConvertCrdFreeformTagsToSdkFreeformTags(&crdObj.Spec.FreeFormTags, &sdkObj.FreeformTags)
	}
	if crdObj.Spec.DefinedTags != nil {
		ConvertCrdDefinedTagsToSdkDefinedTags(&crdObj.Spec.DefinedTags, &sdkObj.DefinedTags)
	}
}

func ConvertCrdAccessPolicyRuleToSdkAccessPolicyRule(crdObj *v1beta1.AccessPolicyRule, sdkObj *sdk.AccessPolicyRule, refIds map[string]api.OCID) {
	sdkObj.Action = sdk.AccessPolicyRuleActionEnum(crdObj.Action)
	sdkObj.Source, _ = ConvertCrdTrafficTargetToSdkAccessPolicyTarget(&crdObj.Source, refIds[meshCommons.Source])
	sdkObj.Destination, _ = ConvertCrdTrafficTargetToSdkAccessPolicyTarget(&crdObj.Destination, refIds[meshCommons.Destination])
}

func ConvertCrdTrafficTargetToSdkAccessPolicyTarget(crdObj *v1beta1.TrafficTarget, refId api.OCID) (sdk.AccessPolicyTarget, error) {

	switch {
	case crdObj.AllVirtualServices != nil:
		return sdk.AllVirtualServicesAccessPolicyTarget{}, nil
	case crdObj.VirtualService != nil:
		vsOcid := string(refId)
		return sdk.VirtualServiceAccessPolicyTarget{VirtualServiceId: &vsOcid}, nil
	case crdObj.ExternalService != nil:
		return convertToSdkExternalService(crdObj.ExternalService)
	case crdObj.IngressGateway != nil:
		igOcid := string(refId)
		return sdk.IngressGatewayAccessPolicyTarget{IngressGatewayId: &igOcid}, nil
	default:
		return nil, errors.New("unknown access policy target")
	}
}

func convertToSdkExternalService(service *v1beta1.ExternalService) (sdk.AccessPolicyTarget, error) {
	switch {
	case service.HttpExternalService != nil:
		return sdk.ExternalServiceAccessPolicyTarget{
			Hostnames: service.HttpExternalService.Hostnames,
			Ports:     convertToSdkPorts(service.HttpExternalService.Ports),
			Protocol:  sdk.ExternalServiceAccessPolicyTargetProtocolHttp,
		}, nil
	case service.HttpsExternalService != nil:
		return sdk.ExternalServiceAccessPolicyTarget{
			Hostnames: service.HttpsExternalService.Hostnames,
			Ports:     convertToSdkPorts(service.HttpsExternalService.Ports),
			Protocol:  sdk.ExternalServiceAccessPolicyTargetProtocolHttps,
		}, nil
	case service.TcpExternalService != nil:
		return sdk.ExternalServiceAccessPolicyTarget{
			IpAddresses: service.TcpExternalService.IpAddresses,
			Ports:       convertToSdkPorts(service.TcpExternalService.Ports),
			Protocol:    sdk.ExternalServiceAccessPolicyTargetProtocolTcp,
		}, nil
	default:
		return nil, fmt.Errorf("invalid external service target %v", service)
	}
}

func convertToSdkPorts(crdPorts []v1beta1.Port) []int {
	if crdPorts == nil {
		return nil
	}
	var sdkPorts = make([]int, len(crdPorts))
	for idx, i := range crdPorts {
		sdkPorts[idx] = int(i)
	}
	return sdkPorts
}

func ConvertSdkAccessPolicyTargetToAccessPolicyTargetDetails(target sdk.AccessPolicyTarget) sdk.AccessPolicyTargetDetails {
	switch target.(type) {
	case sdk.AllVirtualServicesAccessPolicyTarget:
		return sdk.AllVirtualServicesAccessPolicyTargetDetails{}
	case sdk.VirtualServiceAccessPolicyTarget:
		return sdk.VirtualServiceAccessPolicyTargetDetails{VirtualServiceId: target.(sdk.VirtualServiceAccessPolicyTarget).VirtualServiceId}
	case sdk.IngressGatewayAccessPolicyTarget:
		return sdk.IngressGatewayAccessPolicyTargetDetails{IngressGatewayId: target.(sdk.IngressGatewayAccessPolicyTarget).IngressGatewayId}
	}

	return target
}

func ConvertSdkAccessPolicyRuleToSdkAccessPolicyRuleDetails(rules []sdk.AccessPolicyRule) []sdk.AccessPolicyRuleDetails {
	if rules == nil {
		return nil
	}

	ruleDetails := make([]sdk.AccessPolicyRuleDetails, len(rules))
	for i, rule := range rules {
		ruleDetails[i] = sdk.AccessPolicyRuleDetails{
			Action:      sdk.AccessPolicyRuleDetailsActionEnum(rule.Action),
			Source:      ConvertSdkAccessPolicyTargetToAccessPolicyTargetDetails(rule.Source),
			Destination: ConvertSdkAccessPolicyTargetToAccessPolicyTargetDetails(rule.Destination),
		}
	}

	return ruleDetails
}
