/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package ingressgatewaydeployment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/manager"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/validations"
)

//+kubebuilder:webhook:path=/validate-servicemesh-oci-oracle-com-v1beta1-ingressgatewaydeployment,mutating=false,failurePolicy=fail,sideEffects=None,groups=servicemesh.oci.oracle.com,resources=ingressgatewaydeployments,verbs=create;update;delete,versions=v1beta1,name=igd-validator.servicemesh.oci.oracle.cloud.com,admissionReviewVersions={v1,v1beta1}

type IngressGatewayDeploymentValidator struct {
	resolver references.Resolver
	log      loggerutil.OSOKLogger
}

func NewIngressGatewayDeploymentValidator(resolver references.Resolver, log loggerutil.OSOKLogger) manager.CustomResourceValidator {
	return &IngressGatewayDeploymentValidator{resolver: resolver, log: log}
}

func (v *IngressGatewayDeploymentValidator) ValidateOnCreate(context context.Context, object client.Object) (bool, string) {
	igd, err := getIngressGatewayDeployment(object)
	if err != nil {
		return false, err.Error()
	}

	allowed, reason := validations.IsMetadataNameValid(igd.GetName())
	if !allowed {
		return false, reason
	}

	if len(igd.Spec.IngressGateway.Id) == 0 {
		allowed, reason = validations.IsIGPresent(v.resolver, context, igd.Spec.IngressGateway.ResourceRef, &igd.ObjectMeta)
		if !allowed {
			return false, reason
		}
	}

	allowed, reason = v.hasValidPodReplicaConfig(igd)
	if !allowed {
		return false, reason
	}

	allowed, reason = v.hasValidPortConfig(igd)
	if !allowed {
		return false, reason
	}

	allowed, reason = v.hasValidSecretsConfig(igd)
	if !allowed {
		return false, reason
	}

	return true, ""
}

func (v *IngressGatewayDeploymentValidator) GetStatus(object client.Object) (*servicemeshapi.ServiceMeshStatus, error) {
	igd, err := getIngressGatewayDeployment(object)
	if err != nil {
		return nil, err
	}
	return &igd.Status, nil
}

func (v *IngressGatewayDeploymentValidator) ResolveRef(object client.Object) (bool, string) {
	igd, err := getIngressGatewayDeployment(object)
	if err != nil {
		return false, err.Error()
	}
	err = validations.ValidateIGRef(&igd.Spec.IngressGateway)
	if err != nil {
		return false, err.Error()
	}
	return true, ""
}

func (v *IngressGatewayDeploymentValidator) ValidateObject(object client.Object) error {
	_, err := getIngressGatewayDeployment(object)
	return err
}

func (v *IngressGatewayDeploymentValidator) ValidateOnUpdate(context context.Context, object client.Object, oldObject client.Object) (bool, string) {
	igd, err := getIngressGatewayDeployment(object)
	if err != nil {
		return false, err.Error()
	}
	oldIgd, err := getIngressGatewayDeployment(oldObject)
	if err != nil {
		return false, err.Error()
	}

	if !cmp.Equal(igd.Spec.IngressGateway, oldIgd.Spec.IngressGateway) {
		return false, string(commons.IngressGatewayReferenceIsImmutable)
	}

	allowed, reason := v.hasValidPodReplicaConfig(igd)
	if !allowed {
		return false, reason
	}

	allowed, reason = v.hasValidPortConfig(igd)
	if !allowed {
		return false, reason
	}

	allowed, reason = v.hasValidSecretsConfig(igd)
	if !allowed {
		return false, reason
	}

	return true, ""
}

func (v *IngressGatewayDeploymentValidator) hasValidPodReplicaConfig(ingressGatewayDeployment *servicemeshapi.IngressGatewayDeployment) (bool, string) {
	autoscaling := ingressGatewayDeployment.Spec.Deployment.Autoscaling
	if autoscaling.MinPods > autoscaling.MaxPods {
		return false, string(commons.IngressGatewayDeploymentInvalidMaxPod)
	}
	return true, ""
}

func (v *IngressGatewayDeploymentValidator) hasValidPortConfig(ingressGatewayDeployment *servicemeshapi.IngressGatewayDeployment) (bool, string) {
	portMap := make(map[corev1.Protocol][]corev1.ServicePort)
	for _, port := range ingressGatewayDeployment.Spec.Ports {
		if port.ServicePort == nil {
			continue
		}
		if portMap[port.Protocol] == nil {
			portMap[port.Protocol] = make([]corev1.ServicePort, 0)
		}

		portMap[port.Protocol] = append(portMap[port.Protocol], corev1.ServicePort{
			Protocol: port.Protocol,
			Name:     port.Name,
			Port:     *port.ServicePort,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: *port.Port,
			},
		})
	}
	if len(portMap) == 0 {
		return true, ""
	}
	if ingressGatewayDeployment.Spec.Service == nil {
		portsSlice := make([]corev1.ServicePort, 0)
		for _, portsArr := range portMap {
			portsSlice = append(portsSlice, portsArr...)
		}

		if len(portsSlice) > 0 {
			return false, string(commons.IngressGatewayDeploymentRedundantServicePorts)
		}
		return true, ""
	}

	if len(portMap) > 1 {
		return false, string(commons.IngressGatewayDeploymentPortsWithMultipleProtocols)
	}

	// Ensure that the Name of Service Port is unique within the list
	for _, portsArr := range portMap {
		// If single port is specified, name is optional
		if len(portsArr) <= 1 {
			break
		}

		portUniqueNames := make(map[string]bool)
		for _, servicePort := range portsArr {
			if len(servicePort.Name) == 0 {
				return false, string(commons.IngressGatewayDeploymentWithMultiplePortEmptyName)
			}

			if _, exists := portUniqueNames[servicePort.Name]; exists {
				return false, string(commons.IngressGatewayDeploymentPortsWithNonUniqueNames)
			}
			portUniqueNames[servicePort.Name] = true
		}
	}

	return true, ""
}

func (v *IngressGatewayDeploymentValidator) hasValidSecretsConfig(igd *servicemeshapi.IngressGatewayDeployment) (bool, string) {
	secretsMap := make(map[string]bool)
	for _, secret := range igd.Spec.Secrets {
		if secretsMap[secret.SecretName] {
			return false, fmt.Sprintf("spec.ingressgatewaydeployment has duplicate secret %s", secret.SecretName)
		} else {
			secretsMap[secret.SecretName] = true
		}
	}
	return true, ""
}

func getIngressGatewayDeployment(object client.Object) (*servicemeshapi.IngressGatewayDeployment, error) {
	igd, ok := object.(*servicemeshapi.IngressGatewayDeployment)
	if !ok {
		return nil, errors.New("object is not an ingress gateway deployment")
	}
	return igd, nil
}

func (v *IngressGatewayDeploymentValidator) GetEntityType() client.Object {
	return &servicemeshapi.IngressGatewayDeployment{}
}
