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

// ConvertCrdIngressGatewayToSdkIngressGateway converts a CRD object to an object that can be sent to the API
func ConvertCrdIngressGatewayToSdkIngressGateway(crdObj *v1beta1.IngressGateway, sdkObj *sdk.IngressGateway, meshId *api.OCID) {
	sdkObj.Id = (*string)(&crdObj.Status.IngressGatewayId)
	sdkObj.CompartmentId = (*string)(&crdObj.Spec.CompartmentId)
	sdkObj.Name = GetSpecName(crdObj.Spec.Name, &crdObj.ObjectMeta)
	sdkObj.Description = (*string)(crdObj.Spec.Description)
	sdkObj.MeshId = (*string)(meshId)

	sdkObj.Hosts = convertCrdIngressHostToSdkIngressHost(crdObj.Spec.Hosts)
	sdkObj.AccessLogging = convertCrdAccessLoggingToSdkAccessLogging(crdObj.Spec.AccessLogging)

	if crdObj.Spec.FreeFormTags != nil {
		ConvertCrdFreeformTagsToSdkFreeformTags(&crdObj.Spec.FreeFormTags, &sdkObj.FreeformTags)
	}
	if crdObj.Spec.DefinedTags != nil {
		ConvertCrdDefinedTagsToSdkDefinedTags(&crdObj.Spec.DefinedTags, &sdkObj.DefinedTags)
	}
}

func convertCrdIngressHostToSdkIngressHost(crdIngressHosts []v1beta1.IngressGatewayHost) []sdk.IngressGatewayHost {
	sdkIngressHosts := make([]sdk.IngressGatewayHost, 0, len(crdIngressHosts))
	for i, h := range crdIngressHosts {
		sdkIngressHosts = append(sdkIngressHosts, sdk.IngressGatewayHost{
			Name:      (*string)(&crdIngressHosts[i].Name),
			Hostnames: h.Hostnames,
			Listeners: convertCrdIngressGatewayListenerToSdkIngressGatewayListener(h.Listeners),
		})
	}
	return sdkIngressHosts
}

// convertCrdIngressGatewayListenerToSdkIngressGatewayListener converts a listener from a CRD object to a listener for an SDK object
func convertCrdIngressGatewayListenerToSdkIngressGatewayListener(crdListener []v1beta1.IngressGatewayListener) (sdkListeners []sdk.IngressGatewayListener) {
	sdkListeners = make([]sdk.IngressGatewayListener, 0, len(crdListener))
	for _, l := range crdListener {
		sdkListeners = append(sdkListeners, sdk.IngressGatewayListener{
			Protocol: sdk.IngressGatewayListenerProtocolEnum(l.Protocol),
			Port:     PortToInt(&l.Port),
			Tls:      convertCrdIngressGatewayTLSConfigurationToSdkIngressGatewayTlsConfig(l.Tls),
		})
	}
	return sdkListeners
}

// convertCrdIngressGatewayTLSConfigurationToSdkIngressGatewayTlsConfig converts a TLS config from a CRD object to a TLS config for an SDK object
func convertCrdIngressGatewayTLSConfigurationToSdkIngressGatewayTlsConfig(crdObj *v1beta1.IngressListenerTlsConfig) *sdk.IngressListenerTlsConfig {
	if crdObj == nil {
		return nil
	}
	mode := sdk.IngressListenerTlsConfigModeEnum(crdObj.Mode)
	serverCert := convertCrdTlsCertificateToSdkTlsCertificate(crdObj.ServerCertificate)
	clientValidation := convertCrdClientValidationToSdkClientValidation(crdObj.ClientValidation)
	return &sdk.IngressListenerTlsConfig{
		Mode:              mode,
		ServerCertificate: serverCert,
		ClientValidation:  clientValidation,
	}
}

func convertCrdClientValidationToSdkClientValidation(crdObj *v1beta1.IngressHostClientValidationConfig) *sdk.IngressListenerClientValidationConfig {
	if crdObj == nil {
		return nil
	}
	return &sdk.IngressListenerClientValidationConfig{
		TrustedCaBundle:       convertCrdCaBundleToSdkCaBundle(crdObj.TrustedCaBundle),
		SubjectAlternateNames: crdObj.SubjectAlternateNames,
	}
}

func convertCrdCaBundleToSdkCaBundle(crdObj *v1beta1.CaBundle) sdk.CaBundle {
	if crdObj == nil {
		return nil
	}
	switch {
	case crdObj.OciCaBundle != nil:
		return sdk.OciCaBundle{
			CaBundleId: (*string)(&crdObj.OciCaBundle.CaBundleId),
		}
	case crdObj.KubeSecretCaBundle != nil:
		return sdk.LocalFileCaBundle{
			SecretName: &crdObj.KubeSecretCaBundle.SecretName,
		}
	}
	return nil
}

func convertCrdTlsCertificateToSdkTlsCertificate(crdObj *v1beta1.TlsCertificate) sdk.TlsCertificate {
	if crdObj == nil {
		return nil
	}
	switch {
	case crdObj.OciTlsCertificate != nil:
		return sdk.OciTlsCertificate{
			CertificateId: (*string)(&crdObj.OciTlsCertificate.CertificateId),
		}
	case crdObj.KubeSecretTlsCertificate != nil:
		return sdk.LocalFileTlsCertificate{
			SecretName: &crdObj.KubeSecretTlsCertificate.SecretName,
		}
	}
	return nil
}

func ConvertSdkIgMtlsToCrdIgMtls(sdkMtls *sdk.IngressGatewayMutualTransportLayerSecurity) *v1beta1.IngressGatewayMutualTransportLayerSecurity {
	if sdkMtls == nil {
		return nil
	}
	if sdkMtls.CertificateId != nil {
		return &v1beta1.IngressGatewayMutualTransportLayerSecurity{
			CertificateId: *OCID(*sdkMtls.CertificateId),
		}
	}
	return nil
}
