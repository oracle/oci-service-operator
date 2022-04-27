/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package validations

import (
	"context"
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/references"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
)

func IsRefNotUnique(resourceRef *servicemeshapi.ResourceRef, id api.OCID) bool {
	return resourceRef != nil && len(id) > 0
}

func IsRefEmpty(resourceRef *servicemeshapi.ResourceRef, id api.OCID) bool {
	return resourceRef == nil && len(id) == 0
}

func ValidateMeshRef(mesh *servicemeshapi.RefOrId) error {
	if IsRefNotUnique(mesh.ResourceRef, mesh.Id) {
		return errors.New(string(commons.MeshReferenceIsNotUnique))
	}
	if IsRefEmpty(mesh.ResourceRef, mesh.Id) {
		return errors.New(string(commons.MeshReferenceIsEmpty))
	}
	return nil
}

func ValidateVSRef(destination *servicemeshapi.RefOrId) error {
	if IsRefNotUnique(destination.ResourceRef, destination.Id) {
		return errors.New(string(commons.VirtualServiceReferenceIsNotUnique))
	}
	if IsRefEmpty(destination.ResourceRef, destination.Id) {
		return errors.New(string(commons.VirtualServiceReferenceIsEmpty))
	}
	return nil
}

func ValidateVDRef(destination *servicemeshapi.RefOrId) error {
	if IsRefNotUnique(destination.ResourceRef, destination.Id) {
		return errors.New(string(commons.VirtualDeploymentReferenceIsNotUnique))
	}
	if IsRefEmpty(destination.ResourceRef, destination.Id) {
		return errors.New(string(commons.VirtualDeploymentReferenceIsEmpty))
	}
	return nil
}

func ValidateIGRef(destination *servicemeshapi.RefOrId) error {
	if IsRefNotUnique(destination.ResourceRef, destination.Id) {
		return errors.New(string(commons.IngressGatewayReferenceIsNotUnique))
	}
	if IsRefEmpty(destination.ResourceRef, destination.Id) {
		return errors.New(string(commons.IngressGatewayReferenceIsEmpty))
	}
	return nil
}

func IsMeshPresent(resolver references.Resolver, ctx context.Context, resource *servicemeshapi.ResourceRef, objectMeta *metav1.ObjectMeta) (bool, string) {
	resourceRef := resolver.ResolveResourceRef(resource, objectMeta)
	referredMesh, err := resolver.ResolveMeshReference(ctx, resourceRef)
	if err != nil {
		return false, string(commons.MeshReferenceNotFound)
	}
	if !referredMesh.DeletionTimestamp.IsZero() {
		return false, string(commons.MeshReferenceIsDeleting)
	}
	return true, ""
}

func IsVSPresent(resolver references.Resolver, ctx context.Context, resource *servicemeshapi.ResourceRef, objectMeta *metav1.ObjectMeta) (bool, string) {
	resourceRef := resolver.ResolveResourceRef(resource, objectMeta)
	referredVirtualService, err := resolver.ResolveVirtualServiceReference(ctx, resourceRef)
	if err != nil {
		return false, string(commons.VirtualServiceReferenceNotFound)
	}
	if !referredVirtualService.DeletionTimestamp.IsZero() {
		return false, string(commons.VirtualServiceReferenceIsDeleting)
	}
	return true, ""
}

func IsVDPresent(resolver references.Resolver, ctx context.Context, resource *servicemeshapi.ResourceRef, objectMeta *metav1.ObjectMeta) (bool, string) {
	resourceRef := resolver.ResolveResourceRef(resource, objectMeta)
	referredVirtualDeployment, err := resolver.ResolveVirtualDeploymentReference(ctx, resourceRef)
	if err != nil {
		return false, string(commons.VirtualDeploymentReferenceNotFound)
	}
	if !referredVirtualDeployment.DeletionTimestamp.IsZero() {
		return false, string(commons.VirtualDeploymentReferenceIsDeleting)
	}
	return true, ""
}

func IsIGPresent(resolver references.Resolver, ctx context.Context, resource *servicemeshapi.ResourceRef, objectMeta *metav1.ObjectMeta) (bool, string) {
	resourceRef := resolver.ResolveResourceRef(resource, objectMeta)
	referredIngressGateway, err := resolver.ResolveIngressGatewayReference(ctx, resourceRef)
	if err != nil {
		return false, string(commons.IngressGatewayReferenceNotFound)
	}
	if !referredIngressGateway.DeletionTimestamp.IsZero() {
		return false, string(commons.IngressGatewayReferenceIsDeleting)
	}
	return true, ""
}

func IsServicePresent(resolver references.Resolver, ctx context.Context, resource *servicemeshapi.ResourceRef, objectMeta *metav1.ObjectMeta) (bool, string) {
	resourceRef := resolver.ResolveResourceRef(resource, objectMeta)
	referredService, err := resolver.ResolveServiceReferenceWithApiReader(ctx, resourceRef)
	if err != nil {
		return false, string(commons.KubernetesServiceReferenceNotFound)
	}
	if !referredService.DeletionTimestamp.IsZero() {
		return false, string(commons.KubernetesServiceReferenceIsDeleting)
	}
	return true, ""
}

func IsSpecNameChanged(name *servicemeshapi.Name, oldName *servicemeshapi.Name) bool {
	if (name == nil && oldName != nil) || (name != nil && oldName == nil) ||
		(name != nil && oldName != nil && *name != *oldName) {
		return true
	}
	return false
}

func IsMetadataNameValid(name string) (bool, string) {
	if len(name) <= commons.MetadataNameMaxLength {
		return true, ""
	}
	return false, string(commons.MetadataNameLengthExceeded)
}
