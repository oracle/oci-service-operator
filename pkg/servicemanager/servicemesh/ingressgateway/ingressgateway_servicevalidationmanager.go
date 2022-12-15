/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgateway

import (
	"context"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/controller-runtime/pkg/client"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/validations"
)

//+kubebuilder:webhook:path=/validate-servicemesh-oci-oracle-com-v1beta1-ingressgateway,mutating=false,failurePolicy=fail,sideEffects=None,groups=servicemesh.oci.oracle.com,resources=ingressgateways,verbs=create;update;delete,versions=v1beta1,name=ig-validator.servicemesh.oci.oracle.cloud.com,admissionReviewVersions={v1,v1beta1}

type IngressGatewayValidator struct {
	resolver references.Resolver
	log      loggerutil.OSOKLogger
}

func NewIngressGatewayValidator(resolver references.Resolver, log loggerutil.OSOKLogger) manager.CustomResourceValidator {
	return &IngressGatewayValidator{resolver: resolver, log: log}
}

func (v *IngressGatewayValidator) ValidateOnCreate(context context.Context, object client.Object) (bool, string) {
	ig, err := getIngressGateway(object)
	if err != nil {
		return false, err.Error()
	}

	allowed, reason := validations.IsMetadataNameValid(ig.GetName())
	if !allowed {
		return false, reason
	}

	// Only validate for the requests come from k8s operator
	if len(ig.Spec.Mesh.Id) == 0 {
		allowed, reason := validations.IsMeshPresent(v.resolver, context, ig.Spec.Mesh.ResourceRef, &ig.ObjectMeta)
		if !allowed {
			return false, reason
		}
	}

	allowed, reason = v.validateHosts(ig.Spec.Hosts)
	if !allowed {
		return false, reason
	}

	return true, ""
}

func (v *IngressGatewayValidator) GetStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	ig, err := getIngressGateway(object)
	if err != nil {
		return nil, err
	}
	return &ig.Status, nil
}

func (v *IngressGatewayValidator) ResolveRef(object client.Object) (bool, string) {
	ig, err := getIngressGateway(object)
	if err != nil {
		return false, err.Error()
	}
	err = validations.ValidateMeshRef(&ig.Spec.Mesh)
	if err != nil {
		return false, err.Error()
	}
	return true, ""
}

func (v *IngressGatewayValidator) ValidateObject(object client.Object) error {
	_, err := getIngressGateway(object)
	return err
}

func (v *IngressGatewayValidator) ValidateOnUpdate(context context.Context, object client.Object, oldObject client.Object) (bool, string) {
	ig, err := getIngressGateway(object)
	if err != nil {
		return false, err.Error()
	}
	oldIg, err := getIngressGateway(oldObject)
	if err != nil {
		return false, err.Error()
	}

	if !cmp.Equal(ig.Spec.Mesh, oldIg.Spec.Mesh) {
		return false, string(commons.MeshReferenceIsImmutable)
	}

	// Throw an error if name has changed
	if validations.IsSpecNameChanged(ig.Spec.Name, oldIg.Spec.Name) {
		return false, string(commons.NameIsImmutable)
	}

	allowed, reason := v.validateHosts(ig.Spec.Hosts)
	if !allowed {
		return false, reason
	}

	return true, ""
}

func (v *IngressGatewayValidator) validateHosts(hosts []servicemeshapi.IngressGatewayHost) (bool, string) {
	for _, host := range hosts {
		for _, listener := range host.Listeners {
			allowed, reason := validateProtocol(listener.Protocol, host)
			if !allowed {
				return allowed, reason
			}
			allowed, reason = validateListenerPort(listener.Port)
			if !allowed {
				return allowed, reason
			}
			allowed, reason = validateTlsConfig(listener.Tls)
			if !allowed {
				return allowed, reason
			}
		}
	}
	return true, ""
}

func validateProtocol(protocol servicemeshapi.IngressGatewayListenerProtocolEnum, host servicemeshapi.IngressGatewayHost) (bool, string) {
	if protocol == servicemeshapi.IngressGatewayListenerProtocolHttp || protocol == servicemeshapi.IngressGatewayListenerProtocolTlsPassthrough {
		if len(host.Hostnames) == 0 {
			return false, "hostnames is mandatory for a host with HTTP or TLS_PASSTHROUGH listener"
		}
	}
	return true, ""
}

func validateListenerPort(listenerPort servicemeshapi.Port) (bool, string) {
	if listenerPort < 1024 {
		return false, "listener port must be greater than or equal to 1024"
	}
	return true, ""
}

func validateTlsConfig(tlsConfig *servicemeshapi.IngressListenerTlsConfig) (bool, string) {
	if tlsConfig == nil {
		return true, ""
	}

	switch mode := tlsConfig.Mode; mode {
	case servicemeshapi.IngressListenerTlsConfigModeTls, servicemeshapi.IngressListenerTlsConfigModePermissive:
		// validate server cert present
		return validateServerCert(tlsConfig.ServerCertificate)
	case servicemeshapi.IngressListenerTlsConfigModeMutualTls:
		// validate server cert present
		allowed, reason := validateServerCert(tlsConfig.ServerCertificate)
		if !allowed {
			return allowed, reason
		}
		// validate client trustedCaBundle present
		return validateTrustedCaBundle(tlsConfig.ClientValidation)
	}
	return true, ""
}

func validateTrustedCaBundle(clientValidationConfig *servicemeshapi.IngressHostClientValidationConfig) (bool, string) {
	if clientValidationConfig == nil {
		return false, "client validation config is missing"
	}
	if clientValidationConfig.TrustedCaBundle == nil {
		return false, "trusted ca bundle is missing"
	}
	caBundleSourceCount := 0
	if clientValidationConfig.TrustedCaBundle.OciCaBundle != nil {
		caBundleSourceCount++
	}
	if clientValidationConfig.TrustedCaBundle.KubeSecretCaBundle != nil {
		caBundleSourceCount++
	}

	switch {
	case caBundleSourceCount == 0:
		return false, "missing caBundle info"
	case caBundleSourceCount > 1:
		return false, "cannot specify more than 1 caBundle source"
	}

	return true, ""
}

func validateServerCert(certificate *servicemeshapi.TlsCertificate) (bool, string) {
	if certificate == nil {
		return false, "server certificate is missing"
	}
	certSourceCount := 0
	if certificate.OciTlsCertificate != nil {
		certSourceCount++
	}
	if certificate.KubeSecretTlsCertificate != nil {
		certSourceCount++
	}

	switch {
	case certSourceCount == 0:
		return false, "missing certificate info"
	case certSourceCount > 1:
		return false, "cannot specify more than 1 certificate source"
	}

	return true, ""
}

func (v *IngressGatewayValidator) GetEntityType() client.Object {
	return &servicemeshapi.IngressGateway{}
}
