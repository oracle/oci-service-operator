/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package conversions

import (
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"

	"github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
)

func ConvertCrdMeshToSdkMesh(crdMesh *v1beta1.Mesh, sdkMesh *sdk.Mesh) error {
	sdkMesh.Id = (*string)(&crdMesh.Status.MeshId)
	sdkMesh.CompartmentId = (*string)(&crdMesh.Spec.CompartmentId)
	sdkMesh.DisplayName = GetSpecName(crdMesh.Spec.DisplayName, &crdMesh.ObjectMeta)
	sdkMesh.Description = (*string)(crdMesh.Spec.Description)
	sdkMesh.CertificateAuthorities = convertCrdCertificateAuthoritiesToSdkCertificateAuthorities(crdMesh.Spec.CertificateAuthorities)
	var err error
	sdkMesh.Mtls, err = convertCrdMeshMTlsToSdkMeshMTls(crdMesh.Spec.Mtls)
	if err != nil {
		return err
	}
	if crdMesh.Spec.FreeFormTags != nil {
		ConvertCrdFreeformTagsToSdkFreeformTags(&crdMesh.Spec.FreeFormTags, &sdkMesh.FreeformTags)
	}
	if crdMesh.Spec.DefinedTags != nil {
		ConvertCrdDefinedTagsToSdkDefinedTags(&crdMesh.Spec.DefinedTags, &sdkMesh.DefinedTags)
	}
	return nil
}

func convertCrdCertificateAuthoritiesToSdkCertificateAuthorities(crdCertificateAuthorities []v1beta1.CertificateAuthority) []sdk.CertificateAuthority {
	sdkCertificateAuthorities := make([]sdk.CertificateAuthority, 0, len(crdCertificateAuthorities))
	for _, c := range crdCertificateAuthorities {
		sdkCertificateAuthorities = append(sdkCertificateAuthorities, sdk.CertificateAuthority{
			Id: (*string)(&c.Id),
		})
	}
	return sdkCertificateAuthorities
}

func convertCrdMeshMTlsToSdkMeshMTls(crdMtls *v1beta1.MeshMutualTransportLayerSecurity) (*sdk.MeshMutualTransportLayerSecurity, error) {
	if crdMtls == nil {
		return nil, nil
	}
	sdkMtlsMode, err := ConvertCrdMtlsModeEnum(crdMtls.Minimum)
	if err != nil {
		return nil, err
	}
	return &sdk.MeshMutualTransportLayerSecurity{Minimum: sdkMtlsMode}, nil
}

func ConvertSdkMeshMTlsToCrdMeshMTls(sdkMtls *sdk.MeshMutualTransportLayerSecurity) (*v1beta1.MeshMutualTransportLayerSecurity, error) {
	if sdkMtls == nil {
		return nil, nil
	}
	crdMtlsMode, err := ConvertSdkMtlsModeEnum(sdkMtls.Minimum)
	if err != nil {
		return nil, err
	}
	return &v1beta1.MeshMutualTransportLayerSecurity{Minimum: crdMtlsMode}, nil
}
