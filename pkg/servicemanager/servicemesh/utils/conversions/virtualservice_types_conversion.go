/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package conversions

import (
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	"github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

func ConvertCrdVirtualServiceToSdkVirtualService(crdObj *v1beta1.VirtualService, sdkObj *sdk.VirtualService, meshId *api.OCID) error {
	sdkObj.Id = (*string)(&crdObj.Status.VirtualServiceId)
	sdkObj.CompartmentId = (*string)(&crdObj.Spec.CompartmentId)
	sdkObj.Name = GetSpecName(crdObj.Spec.Name, &crdObj.ObjectMeta)
	sdkObj.Description = (*string)(crdObj.Spec.Description)
	sdkObj.MeshId = (*string)(meshId)
	sdkObj.Hosts = crdObj.Spec.Hosts

	if crdObj.Spec.DefaultRoutingPolicy != nil {
		sdkObj.DefaultRoutingPolicy = &sdk.DefaultVirtualServiceRoutingPolicy{
			Type: sdk.DefaultVirtualServiceRoutingPolicyTypeEnum(crdObj.Spec.DefaultRoutingPolicy.Type),
		}
	}
	var err error
	sdkObj.Mtls, err = convertCrdVsMtlsToSdkVsMtls(crdObj.Spec.Mtls)
	if err != nil {
		return err
	}

	if crdObj.Spec.FreeFormTags != nil {
		sdkObj.FreeformTags = crdObj.Spec.FreeFormTags
	}
	if crdObj.Spec.DefinedTags != nil {
		sdkObj.DefinedTags = map[string]map[string]interface{}{}
		ConvertCrdDefinedTagsToSdkDefinedTags(&crdObj.Spec.DefinedTags, &sdkObj.DefinedTags)
	}
	return nil
}

func convertCrdVsMtlsToSdkVsMtls(crdMtls *v1beta1.CreateVirtualServiceMutualTransportLayerSecurity) (*sdk.MutualTransportLayerSecurity, error) {
	if crdMtls == nil {
		return nil, nil
	}
	sdkMtls, err := ConvertCrdMtlsModeEnum(crdMtls.Mode)
	if err != nil {
		return nil, err
	}
	return &sdk.MutualTransportLayerSecurity{Mode: sdkMtls}, nil
}

func ConvertSdkVsMtlsToCrdVsMtls(sdkMtls *sdk.MutualTransportLayerSecurity) (*v1beta1.VirtualServiceMutualTransportLayerSecurity, error) {
	if sdkMtls == nil {
		return nil, nil
	}
	sdkMtlsMode, err := ConvertSdkMtlsModeEnum(sdkMtls.Mode)
	if err != nil {
		return nil, err
	}

	crdMtls := &v1beta1.VirtualServiceMutualTransportLayerSecurity{
		Mode: sdkMtlsMode,
	}
	if sdkMtls.CertificateId != nil {
		crdMtls.CertificateId = OCID(*sdkMtls.CertificateId)
	}
	return crdMtls, nil
}
