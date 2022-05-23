/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package validator

import (
	sdk "github.com/oracle/oci-go-sdk/v65/servicemesh"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	meshErrors "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/errors"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidateMeshK8s checks if a mesh is valid else returns errors
func ValidateMeshK8s(mesh *servicemeshapi.Mesh) error {
	if mesh.Status.Conditions == nil || len(mesh.Status.Conditions) != 3 {
		return meshErrors.NewRequeueOnError(errors.Errorf("mesh status condition is not yet satisfied"))
	}
	for _, condition := range mesh.Status.Conditions {
		if condition.Status != metav1.ConditionTrue {
			return meshErrors.NewRequeueOnError(errors.Errorf("mesh status condition %s is not yet satisfied", condition.Type))
		}
	}
	return nil
}

// ValidateMeshCp checks if a mesh is valid else returns errors
func ValidateMeshCp(mesh *sdk.Mesh) error {
	if mesh.LifecycleState == sdk.MeshLifecycleStateDeleted ||
		mesh.LifecycleState == sdk.MeshLifecycleStateFailed {
		return meshErrors.NewDoNotRequeueError(errors.New("mesh is deleted or failed"))
	}
	if mesh.LifecycleState != sdk.MeshLifecycleStateActive {
		return meshErrors.NewRequeueOnError(errors.New("mesh is not active yet"))
	}
	return nil
}

// ValidateVSK8s checks if a VS is valid else returns errors
func ValidateVSK8s(virtualService *servicemeshapi.VirtualService) error {
	if virtualService.Status.Conditions == nil || len(virtualService.Status.Conditions) != 3 {
		return meshErrors.NewRequeueOnError(errors.Errorf("virtual service status condition is not yet satisfied"))
	}
	for _, condition := range virtualService.Status.Conditions {
		if condition.Status != metav1.ConditionTrue {
			return meshErrors.NewRequeueOnError(errors.Errorf("virtual service status condition %s is not yet satisfied", condition.Type))
		}
	}
	return nil
}

// ValidateVSCp checks if a VS is valid else returns errors
func ValidateVSCp(virtualService *sdk.VirtualService) error {
	if virtualService.LifecycleState == sdk.VirtualServiceLifecycleStateDeleted ||
		virtualService.LifecycleState == sdk.VirtualServiceLifecycleStateFailed {
		return meshErrors.NewDoNotRequeueError(errors.New("virtual service is deleted or failed"))
	}
	if virtualService.LifecycleState != sdk.VirtualServiceLifecycleStateActive {
		return meshErrors.NewRequeueOnError(errors.New("virtual service is not active yet"))
	}
	return nil
}

// ValidateVDK8s checks if a VD is valid else returns errors
func ValidateVDK8s(vd *servicemeshapi.VirtualDeployment) error {
	if vd.Status.Conditions == nil || len(vd.Status.Conditions) != 3 {
		return meshErrors.NewRequeueOnError(errors.Errorf("virtual deployment status condition is not yet satisfied"))
	}
	for _, condition := range vd.Status.Conditions {
		if condition.Status != metav1.ConditionTrue {
			return meshErrors.NewRequeueOnError(errors.Errorf("virtual deployment status condition %s is not yet satisfied", condition.Type))
		}
	}
	return nil
}

// ValidateVDCp checks if a VD is valid else returns errors
func ValidateVDCp(vd *sdk.VirtualDeployment) error {
	if vd.LifecycleState == sdk.VirtualDeploymentLifecycleStateDeleted ||
		vd.LifecycleState == sdk.VirtualDeploymentLifecycleStateFailed {
		return meshErrors.NewDoNotRequeueError(errors.New("virtual deployment is deleted or failed"))
	}
	if vd.LifecycleState != sdk.VirtualDeploymentLifecycleStateActive {
		return meshErrors.NewRequeueOnError(errors.New("virtual deployment is not active yet"))
	}
	return nil
}

// ValidateIGK8s checks if a IG is valid else returns errors
func ValidateIGK8s(ingressGateway *servicemeshapi.IngressGateway) error {
	if ingressGateway.Status.Conditions == nil || len(ingressGateway.Status.Conditions) != 3 {
		return meshErrors.NewRequeueOnError(errors.Errorf("ingress gateway status condition is not yet satisfied"))
	}
	for _, condition := range ingressGateway.Status.Conditions {
		if condition.Status != metav1.ConditionTrue {
			return meshErrors.NewRequeueOnError(errors.Errorf("ingress gateway status condition %s is not yet satisfied", condition.Type))
		}
	}
	return nil
}

// ValidateIGCp checks if a IG is valid else returns errors
func ValidateIGCp(ingressGateway *sdk.IngressGateway) error {
	if ingressGateway.LifecycleState == sdk.IngressGatewayLifecycleStateDeleted ||
		ingressGateway.LifecycleState == sdk.IngressGatewayLifecycleStateFailed {
		return meshErrors.NewDoNotRequeueError(errors.New("ingress gateway is deleted or failed"))
	}
	if ingressGateway.LifecycleState != sdk.IngressGatewayLifecycleStateActive {
		return meshErrors.NewRequeueOnError(errors.New("ingress gateway is not active yet"))
	}
	return nil
}
