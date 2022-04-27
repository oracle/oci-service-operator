/*
  Copyright (c) 2022, Oracle and/or its affiliates. All rights reserved.
  Licensed under the Universal Permissive License v 1.0 as shown at http://oss.oracle.com/licenses/upl.
*/

package references

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/oracle/oci-service-operator/api/v1beta1"
	servicemeshapi "github.com/oracle/oci-service-operator/apis/servicemesh.oci/v1beta1"
	"github.com/oracle/oci-service-operator/pkg/loggerutil"
	customCache "github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/k8s/cache"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/services"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/commons"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/conversions"
	"github.com/oracle/oci-service-operator/pkg/servicemanager/servicemesh/utils/validator"
)

type Resolver interface {
	ResolveResourceRef(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef
	// ResolveMeshReference returns a mesh CR based on ref
	ResolveMeshReference(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error)
	// ResolveMeshId returns MehId for a given RefOrId
	ResolveMeshId(ctx context.Context, meshRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error)
	// ResolveMeshRefById returns MeshRef for a given meshId
	ResolveMeshRefById(ctx context.Context, meshId *api.OCID) (*commons.MeshRef, error)
	// ResolveVirtualServiceReference returns a virtual service CR based on ref
	ResolveVirtualServiceReference(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error)
	// ResolveVirtualServiceIdAndName returns ResourceRef for a given RefOrId
	ResolveVirtualServiceIdAndName(ctx context.Context, virtualServiceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
	// ResolveVirtualServiceRefById returns VirtualServiceRef for a given virtualServiceId
	ResolveVirtualServiceRefById(ctx context.Context, virtualServiceId *api.OCID) (*commons.ResourceRef, error)
	// ResolveVirtualServiceList resolves all the virtual services under a given namespace
	ResolveVirtualServiceListByNamespace(ctx context.Context, namespace string) (servicemeshapi.VirtualServiceList, error)
	// ResolveVirtualDeploymentReference returns a virtual deployment CR based on ref
	ResolveVirtualDeploymentReference(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error)
	// ResolveVirtualDeploymentId returns VirtualDeploymentId for a given VirtualDeploymentRefOrId
	ResolveVirtualDeploymentId(ctx context.Context, virtualDeploymentRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error)
	// ResolveIngressGatewayReference returns a ingress gateway CR based on ref
	ResolveIngressGatewayReference(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error)
	// ResolveIngressGatewayIdAndNameAndMeshId returns IngressGatewayRef for a given IngressGatewayRefOrId
	ResolveIngressGatewayIdAndNameAndMeshId(ctx context.Context, IngressGatewayRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error)
	// ResolveServiceReference returns a k8s service based on ref from Cache
	ResolveServiceReference(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error)
	// ResolveServiceReferenceWithApiReader returns a k8s service based on ref from K8s API Server
	ResolveServiceReferenceWithApiReader(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error)
}

// defaultResolver implements Resolver
type defaultResolver struct {
	client       client.Client
	cache        customCache.CacheMapClient
	meshClient   services.ServiceMeshClient
	log          loggerutil.OSOKLogger
	directReader client.Reader // direct reader reads data from api server
}

// NewDefaultResolver constructs new defaultResolver
func NewDefaultResolver(k8sClient client.Client, meshClient services.ServiceMeshClient, log loggerutil.OSOKLogger, cache customCache.CacheMapClient, k8sDirectReader client.Reader) Resolver {
	return &defaultResolver{
		client:       k8sClient,
		meshClient:   meshClient,
		log:          log,
		cache:        cache,
		directReader: k8sDirectReader,
	}
}

func (r *defaultResolver) ResolveMeshId(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
	meshId := api.OCID("")
	if len(resourceRef.Id) > 0 {
		meshCr, err := r.meshClient.GetMesh(ctx, &resourceRef.Id)
		if err != nil {
			return nil, err
		}
		meshId = api.OCID(*meshCr.Id)
		if err = validator.ValidateMeshCp(meshCr); err != nil {
			return &meshId, err
		}
	} else {
		refObj := r.ResolveResourceRef(resourceRef.ResourceRef, crdObj)
		mesh, err := r.ResolveMeshReference(ctx, refObj)
		if err != nil {
			return nil, err
		}
		if !mesh.DeletionTimestamp.IsZero() {
			return nil, kerrors.NewResourceExpired(fmt.Sprintf("referenced mesh object with name: %s and namespace: %s is marked for deletion", refObj.Name, refObj.Namespace))
		}
		if err = validator.ValidateMeshK8s(mesh); err != nil {
			return nil, err
		}
		meshId = mesh.Status.MeshId
	}
	return &meshId, nil
}

func (r *defaultResolver) ResolveMeshReference(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.Mesh, error) {
	mesh := &servicemeshapi.Mesh{}
	if err := r.directReader.Get(ctx, types.NamespacedName{Namespace: ref.Namespace, Name: string(ref.Name)}, mesh); err != nil {
		r.log.ErrorLog(err, "unable to fetch mesh", "name", string(ref.Name), "namespace", ref.Namespace)
		return nil, err
	}
	return mesh, nil
}

func (r *defaultResolver) ResolveVirtualServiceIdAndName(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
	virtualServiceRef := commons.ResourceRef{
		Id:     api.OCID(""),
		Name:   servicemeshapi.Name(""),
		MeshId: api.OCID(""),
	}
	if len(resourceRef.Id) > 0 {
		virtualServiceCp, err := r.meshClient.GetVirtualService(ctx, &resourceRef.Id)
		if err != nil {
			return nil, err
		}
		virtualServiceRef.Id = api.OCID(*virtualServiceCp.Id)
		virtualServiceRef.Name = servicemeshapi.Name(*virtualServiceCp.Name)
		virtualServiceRef.MeshId = api.OCID(*virtualServiceCp.MeshId)
		if err = validator.ValidateVSCp(virtualServiceCp); err != nil {
			return &virtualServiceRef, err
		}
	} else {
		refObj := r.ResolveResourceRef(resourceRef.ResourceRef, crdObj)
		virtualServiceCr, err := r.ResolveVirtualServiceReference(ctx, refObj)
		if err != nil {
			return nil, err
		}
		if !virtualServiceCr.DeletionTimestamp.IsZero() {
			return nil, kerrors.NewResourceExpired(fmt.Sprintf("referenced virtual service object with name: %s and namespace: %s is marked for deletion", refObj.Name, refObj.Namespace))
		}
		if err = validator.ValidateVSK8s(virtualServiceCr); err != nil {
			return nil, err
		}
		virtualServiceRef.Id = virtualServiceCr.Status.VirtualServiceId
		virtualServiceRef.Name = servicemeshapi.Name(*conversions.GetSpecName(virtualServiceCr.Spec.Name, &virtualServiceCr.ObjectMeta))
		virtualServiceRef.MeshId = virtualServiceCr.Status.MeshId
	}
	return &virtualServiceRef, nil
}

func (r *defaultResolver) ResolveVirtualServiceReference(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualService, error) {
	virtualService := &servicemeshapi.VirtualService{}
	if err := r.directReader.Get(ctx, types.NamespacedName{Namespace: ref.Namespace, Name: string(ref.Name)}, virtualService); err != nil {
		r.log.ErrorLog(err, "unable to fetch virtual service", "name", string(ref.Name), "namespace", ref.Namespace)
		return nil, err
	}

	return virtualService, nil
}

func (r *defaultResolver) ResolveVirtualDeploymentId(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*api.OCID, error) {
	virtualDeploymentId := api.OCID("")
	if len(resourceRef.Id) > 0 {
		virtualDeploymentCp, err := r.meshClient.GetVirtualDeployment(ctx, &resourceRef.Id)
		if err != nil {
			return nil, err
		}
		virtualDeploymentId = api.OCID(*virtualDeploymentCp.Id)
		if err = validator.ValidateVDCp(virtualDeploymentCp); err != nil {
			return &virtualDeploymentId, err
		}
	} else {
		refObj := r.ResolveResourceRef(resourceRef.ResourceRef, crdObj)
		virtualDeploymentCr, err := r.ResolveVirtualDeploymentReference(ctx, refObj)
		if err != nil {
			return nil, err
		}
		if !virtualDeploymentCr.DeletionTimestamp.IsZero() {
			return nil, kerrors.NewResourceExpired(fmt.Sprintf("referenced virtual deployment object with name: %s and namespace: %s is marked for deletion", refObj.Name, refObj.Namespace))
		}
		if err = validator.ValidateVDK8s(virtualDeploymentCr); err != nil {
			return nil, err
		}
		virtualDeploymentId = virtualDeploymentCr.Status.VirtualDeploymentId
	}
	return &virtualDeploymentId, nil
}

func (r *defaultResolver) ResolveVirtualDeploymentReference(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.VirtualDeployment, error) {
	virtualDeployment := &servicemeshapi.VirtualDeployment{}
	if err := r.directReader.Get(ctx, types.NamespacedName{Namespace: ref.Namespace, Name: string(ref.Name)}, virtualDeployment); err != nil {
		r.log.ErrorLog(err, "unable to fetch virtual deployment", "name", string(ref.Name), "namespace", ref.Namespace)
		return nil, err
	}
	return virtualDeployment, nil
}

func (r *defaultResolver) ResolveIngressGatewayIdAndNameAndMeshId(ctx context.Context, resourceRef *servicemeshapi.RefOrId, crdObj *metav1.ObjectMeta) (*commons.ResourceRef, error) {
	ingressGatewayRef := commons.ResourceRef{
		Id:     api.OCID(""),
		Name:   servicemeshapi.Name(""),
		MeshId: api.OCID(""),
	}
	if len(resourceRef.Id) > 0 {
		ingressGatewayCp, err := r.meshClient.GetIngressGateway(ctx, &resourceRef.Id)
		if err != nil {
			return nil, err
		}
		ingressGatewayRef.Id = api.OCID(*ingressGatewayCp.Id)
		ingressGatewayRef.Name = servicemeshapi.Name(*ingressGatewayCp.Name)
		ingressGatewayRef.MeshId = api.OCID(*ingressGatewayCp.MeshId)
		if err = validator.ValidateIGCp(ingressGatewayCp); err != nil {
			return &ingressGatewayRef, err
		}
	} else {
		refObj := r.ResolveResourceRef(resourceRef.ResourceRef, crdObj)
		ingressGatewayCr, err := r.ResolveIngressGatewayReference(ctx, refObj)
		if err != nil {
			return nil, err
		}
		if !ingressGatewayCr.DeletionTimestamp.IsZero() {
			return nil, kerrors.NewResourceExpired(fmt.Sprintf("referenced ingress gateway object with name: %s and namespace: %s is marked for deletion", refObj.Name, refObj.Namespace))
		}
		if err = validator.ValidateIGK8s(ingressGatewayCr); err != nil {
			return nil, err
		}
		ingressGatewayRef.Id = ingressGatewayCr.Status.IngressGatewayId
		ingressGatewayRef.Name = servicemeshapi.Name(*conversions.GetSpecName(ingressGatewayCr.Spec.Name, &ingressGatewayCr.ObjectMeta))
		ingressGatewayRef.MeshId = ingressGatewayCr.Status.MeshId
	}
	return &ingressGatewayRef, nil
}

func (r *defaultResolver) ResolveIngressGatewayReference(ctx context.Context, ref *servicemeshapi.ResourceRef) (*servicemeshapi.IngressGateway, error) {
	ingressGateway := &servicemeshapi.IngressGateway{}
	if err := r.directReader.Get(ctx, types.NamespacedName{Namespace: ref.Namespace, Name: string(ref.Name)}, ingressGateway); err != nil {
		r.log.ErrorLog(err, "unable to fetch ingress gateway", "name", string(ref.Name), "namespace", ref.Namespace)
		return nil, err
	}

	return ingressGateway, nil
}

func (r *defaultResolver) ResolveMeshRefById(ctx context.Context, meshId *api.OCID) (*commons.MeshRef, error) {
	meshCp, err := r.meshClient.GetMesh(ctx, meshId)
	if err != nil {
		return nil, err
	}
	crdMtls, err := conversions.ConvertSdkMeshMTlsToCrdMeshMTls(meshCp.Mtls)
	if err != nil {
		return nil, err
	}
	meshRef := commons.MeshRef{
		Id:          *meshId,
		DisplayName: servicemeshapi.Name(*meshCp.DisplayName),
		Mtls:        *crdMtls,
	}
	return &meshRef, nil
}

func (r *defaultResolver) ResolveVirtualServiceListByNamespace(ctx context.Context, namespace string) (servicemeshapi.VirtualServiceList, error) {
	var listOptions client.ListOptions
	listOptions.Namespace = namespace
	virtualServiceList := &servicemeshapi.VirtualServiceList{}
	if err := r.directReader.List(ctx, virtualServiceList, &listOptions); err != nil {
		r.log.ErrorLog(err, "unable to fetch virtual service list for ", "namespace", namespace)
		return servicemeshapi.VirtualServiceList{}, err
	}
	return *virtualServiceList, nil
}

func (r *defaultResolver) ResolveVirtualServiceRefById(ctx context.Context, virtualServiceId *api.OCID) (*commons.ResourceRef, error) {
	virtualServiceCp, err := r.meshClient.GetVirtualService(ctx, virtualServiceId)
	if err != nil {
		return nil, err
	}
	virtualServiceRef := commons.ResourceRef{
		Id:     *virtualServiceId,
		Name:   servicemeshapi.Name(*virtualServiceCp.Name),
		MeshId: api.OCID(*virtualServiceCp.MeshId),
	}
	return &virtualServiceRef, nil
}

func (r *defaultResolver) ResolveServiceReference(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
	service, err := r.cache.GetServiceByKey(ref.Namespace + "/" + string(ref.Name))
	if err != nil {
		r.log.ErrorLog(err, "unable to fetch service", "name", string(ref.Name), "namespace", ref.Namespace)
		return nil, err
	}
	return service, nil
}

func (r *defaultResolver) ResolveServiceReferenceWithApiReader(ctx context.Context, ref *servicemeshapi.ResourceRef) (*corev1.Service, error) {
	service := &corev1.Service{}
	if err := r.directReader.Get(ctx, types.NamespacedName{Namespace: ref.Namespace, Name: string(ref.Name)}, service); err != nil {
		r.log.ErrorLog(err, "unable to fetch service", "name", string(ref.Name), "namespace", ref.Namespace)
		return nil, err

	}
	return service, nil
}

func (r *defaultResolver) ResolveResourceRef(resourceRef *servicemeshapi.ResourceRef, crdObj *metav1.ObjectMeta) *servicemeshapi.ResourceRef {
	refObj := &servicemeshapi.ResourceRef{
		Name:      resourceRef.Name,
		Namespace: resourceRef.Namespace,
	}
	if resourceRef.Namespace == "" {
		refObj.Namespace = crdObj.Namespace
	}
	return refObj
}
